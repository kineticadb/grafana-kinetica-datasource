package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/hamba/avro/v2"
	"github.com/kineticadb/kinetica-api-go/kinetica"
	"golang.org/x/sync/errgroup"
)

var (
	_ backend.QueryDataHandler    = (*Datasource)(nil)
	_ backend.CheckHealthHandler  = (*Datasource)(nil)
	_ backend.CallResourceHandler = (*Datasource)(nil)
)

type Datasource struct {
	client *kinetica.Kinetica
}

func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	config, err := loadSettings(&settings)
	if err != nil {
		logError("NewDatasource: failed to load settings", "err", err)
		return nil, err
	}
	client := createClient(ctx, config, &settings)
	return &Datasource{client: client}, nil
}

// -------------------------------------------------------------------------
// 1. QUERY DATA
// -------------------------------------------------------------------------
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	// For a single query, skip the overhead of goroutines
	if len(req.Queries) == 1 {
		res := d.query(ctx, d.client, req.Queries[0])
		response.Responses[req.Queries[0].RefID] = res
		return response, nil
	}

	// Run multiple queries concurrently using errgroup
	g, gCtx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	results := make(map[string]backend.DataResponse)

	for _, q := range req.Queries {
		// Capture loop variable for goroutine
		query := q
		g.Go(func() error {
			res := d.query(gCtx, d.client, query)
			mu.Lock()
			results[query.RefID] = res
			mu.Unlock()
			return nil // Don't propagate errors to cancel other queries
		})
	}

	// Wait for all queries to complete
	_ = g.Wait()

	// Collect results
	for refID, res := range results {
		response.Responses[refID] = res
	}

	return response, nil
}

func (d *Datasource) query(ctx context.Context, db *kinetica.Kinetica, query backend.DataQuery) backend.DataResponse {

	logDebug("Starting query processing", "refId", query.RefID)

	// 1. Parse Query Model
	var qModel QueryModel
	if err := json.Unmarshal(query.JSON, &qModel); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "Failed to unmarshal query model")
	}

	// 2. Prepare SQL (Macros)
	sql, err := prepareSql(ctx, query, db)
	if err != nil {
		logError("Macro expansion failed", "refId", query.RefID, "err", err)
		return backend.ErrDataResponse(backend.StatusBadRequest, "Query failed, see Grafana server log for details")
	}
	if sql == "" {
		return backend.DataResponse{}
	}

	logDebug("Macro replacement complete", "refId", query.RefID, "final_sql", sql)

	// 3. Execute SQL
	logDebug("Sending query to Kinetica", "refId", query.RefID)
	options := &kinetica.ExecuteSqlOptions{Encoding: "binary"}
	rawResp, err := db.ExecuteSqlRawWithOpts(ctx, sql, 0, -9999, "", nil, options)
	if err != nil {
		logError("Kinetica query execution failed", "refId", query.RefID, "err", err)
		return backend.ErrDataResponse(backend.StatusBadGateway, "Query failed, see Grafana server log for details")
	}

	// 4. Parse Avro Schema
	schema, err := avro.Parse(rawResp.ResponseSchema)
	if err != nil {
		logError("Avro schema parse failed", "refId", query.RefID, "err", err)
		return backend.ErrDataResponse(backend.StatusInternal, "Query failed, see Grafana server log for details")
	}

	// 5. Unmarshal Data
	rawMap := make(map[string]any)
	err = avro.Unmarshal(schema, rawResp.BinaryEncodedResponse, &rawMap)
	if err != nil {
		logError("Avro unmarshal failed", "refId", query.RefID, "err", err)
		return backend.ErrDataResponse(backend.StatusInternal, "Query failed, see Grafana server log for details")
	}

	// 6. Columns, in the order the query selects them, with their Kinetica types
	columns, ok := resultColumns(rawMap)
	if !ok {
		return backend.DataResponse{Frames: data.Frames{data.NewFrame("response")}}
	}

	// 7. Time columns. The response names each column's type, which identifies
	// the time columns whatever their alias or the query's shape; the queried
	// table's metadata is only read when the server sends no types.
	timeCols := make(map[string]bool)
	if !hasDatatypes(columns) {
		timeCols = d.metadataTimeColumns(ctx, db, qModel, sql)
	}
	// Aliases produced by $__timeGroup, read from the pre-expansion SQL, where the
	// macro is still recognisable
	for _, alias := range timeGroupAliases(qModel.RawSql) {
		timeCols[alias] = true
	}

	// 8. Convert to Frame
	frame, err := parseToFrame(columns, timeCols, query.RefID)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, err.Error())
	}

	// 9. Time series format: one series per combination of the string columns
	if qModel.Format == formatTimeSeries {
		frame, err = toTimeSeriesFrame(frame)
		if err != nil {
			logError("Time series conversion failed", "refId", query.RefID, "err", err)
			return backend.ErrDataResponse(backend.StatusBadRequest, "Query failed, see Grafana server log for details")
		}
	}

	return backend.DataResponse{Frames: data.Frames{frame}}
}

// metadataTimeColumns - the time columns of a query read from the metadata of
// the table it queries: the builder's table, or else the first table named
// after a FROM; for servers whose responses carry no column types
//
//	@return map[string]bool - the names of the time columns, with the builder's
//	aliases of them
func (d *Datasource) metadataTimeColumns(ctx context.Context, db *kinetica.Kinetica, qModel QueryModel, sql string) map[string]bool {
	// A. Try Builder
	fullTableName := qModel.Builder.Table
	if qModel.Builder.Schema != "" && !strings.HasPrefix(qModel.Builder.Table, qModel.Builder.Schema+".") {
		fullTableName = fmt.Sprintf("%s.%s", qModel.Builder.Schema, qModel.Builder.Table)
	}

	// B. Fallback: Raw SQL
	if fullTableName == "" || fullTableName == "." {
		fullTableName = fromTableName(sql)
		logDebug("Table name extracted from Raw SQL", "table", fullTableName)
	}

	timeCols, err := d.getTimeColumns(ctx, db, fullTableName)
	if err != nil {
		logWarn("Could not fetch table metadata", "table", fullTableName, "err", err)
	}
	if timeCols == nil {
		timeCols = make(map[string]bool)
	}

	// Map the builder's aliases of time columns, so parseToFrame knows them
	for _, sel := range qModel.Builder.Selects {
		if sel.Alias == "" || sel.Column == "" {
			continue
		}
		// Clean quotes sent by the frontend
		cleanCol := strings.Trim(sel.Column, "\"")
		cleanAlias := strings.Trim(sel.Alias, "\"")
		if !timeCols[cleanCol] {
			continue
		}
		// Only preserve the time type if the function preserves time (MIN, MAX,
		// or none): COUNT, SUM, AVG of a date returns a number, not a date
		agg := strings.ToUpper(sel.Aggregate)
		if agg == "" || agg == "MIN" || agg == "MAX" || agg == "FIRST" || agg == "LAST" {
			timeCols[cleanAlias] = true
		}
	}
	return timeCols
}

// fromTableName - the first table named after a FROM in a query, without
// quotes: FROM table, FROM schema.table or FROM "schema"."table"; "" if none
func fromTableName(sql string) string {
	match := fromTablePattern.FindStringSubmatch(sql)
	if len(match) < 2 {
		return ""
	}
	return strings.ReplaceAll(match[1], "\"", "")
}

var fromTablePattern = regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z0-9_."]+)`)

// -------------------------------------------------------------------------
// 2. METADATA & MACRO HELPERS
// -------------------------------------------------------------------------

func (d *Datasource) getTimeColumns(ctx context.Context, db *kinetica.Kinetica, tableName string) (map[string]bool, error) {
	if tableName == "" || tableName == "." {
		return nil, nil
	}

	resp, err := db.ShowTableRawWithOpts(ctx, tableName, &kinetica.ShowTableOptions{GetColumnInfo: true})
	if err != nil {
		return nil, err
	}

	if len(resp.Properties) == 0 {
		return nil, nil
	}

	columnProps := resp.Properties[0]
	timeCols := make(map[string]bool)

	for colName, props := range columnProps {
		for _, prop := range props {
			if prop == "datetime" || prop == "timestamp" || prop == "date" {
				timeCols[colName] = true
				break
			}
		}
	}
	return timeCols, nil
}

// parseIntervalMillis converts a Grafana interval literal into milliseconds.
// Accepts "30s", "5m", "1h", "7d", "2w" and "500ms", with or without surrounding
// quotes, since users write $__timeGroup(ts, '5m') as well as
// $__timeGroup(ts, $__interval). A bare number is taken as milliseconds, matching
// what Grafana substitutes for $__interval_ms.

// timeGroupAliases returns the aliases assigned to $__timeGroup(...) expressions in
// a query, e.g. "time" for `$__timeGroup(ts, 5m) AS time`. Used to mark those output
// columns as time columns, which their name alone cannot reveal.
func timeGroupAliases(sql string) []string {
	re := regexp.MustCompile(`(?i)\$__timeGroup\([^()]*\)\s+AS\s+"?([A-Za-z0-9_]+)"?`)
	var aliases []string
	for _, m := range re.FindAllStringSubmatch(sql, -1) {
		if len(m) > 1 {
			aliases = append(aliases, m[1])
		}
	}
	return aliases
}

func parseIntervalMillis(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "'\"")
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("$__timeGroup: empty interval")
	}

	// "ms" must be tested before "s", and "m" only after both.
	units := []struct {
		suffix string
		millis int64
	}{
		{"ms", 1},
		{"s", 1000},
		{"m", 60 * 1000},
		{"h", 60 * 60 * 1000},
		{"d", 24 * 60 * 60 * 1000},
		{"w", 7 * 24 * 60 * 60 * 1000},
	}

	for _, u := range units {
		if !strings.HasSuffix(s, u.suffix) {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(s, u.suffix)), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("$__timeGroup: invalid interval %q", raw)
		}
		if n <= 0 {
			return 0, fmt.Errorf("$__timeGroup: interval %q must be positive", raw)
		}
		return n * u.millis, nil
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("$__timeGroup: invalid interval %q", raw)
	}
	if n <= 0 {
		return 0, fmt.Errorf("$__timeGroup: interval %q must be positive", raw)
	}
	return n, nil
}

// timeGroupExpr builds SQL rounding a time value down to an interval boundary.
//
// TIME_BUCKET(width_ms, ts) is Kinetica's native bucketing function and takes a
// bare numeric width in milliseconds, which is the unit parseIntervalMillis already
// produces. It returns a datetime, so time-of-day survives and sub-second widths work.
//
// The column is wrapped in TIMESTAMP() to make this work for every column kind:
// passed a raw epoch-milliseconds number, TIME_BUCKET reads it as a time-of-day and
// silently loses the date. TIMESTAMP() is a no-op for columns that are already
// timestamps and parses string datetimes, so one form covers all three.
func timeGroupExpr(colName string, millis int64) string {
	return fmt.Sprintf("TIME_BUCKET(%d, TIMESTAMP(%s))", millis, colName)
}

func prepareSql(ctx context.Context, q backend.DataQuery, db *kinetica.Kinetica) (string, error) {
	var qModel QueryModel
	if err := json.Unmarshal(q.JSON, &qModel); err != nil {
		return "", err
	}
	sql := qModel.RawSql

	// 1. $__timeFilter(col)
	reTimeFilter := regexp.MustCompile(`\$__timeFilter\((.*?)\)`)
	sql = reTimeFilter.ReplaceAllStringFunc(sql, func(match string) string {
		submatches := reTimeFilter.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		colName := submatches[1]

		// The column's table: the builder's, or for raw SQL the first table
		// named after a FROM
		schema, table := qModel.Builder.Schema, qModel.Builder.Table
		if table == "" {
			schema, table = "", fromTableName(qModel.RawSql)
		}
		isString, err := getColumnIsString(ctx, db, schema, table, colName)
		if err != nil {
			logError("Failed to lookup column type", "err", err)
			isString = false
		}

		if isString {
			fromStr := kineticaTimeLiteral(q.TimeRange.From)
			toStr := kineticaTimeLiteral(q.TimeRange.To)
			return fmt.Sprintf("%s >= %s AND %s <= %s", colName, fromStr, colName, toStr)
		} else {
			return fmt.Sprintf("%s >= %d AND %s <= %d", colName, q.TimeRange.From.UnixMilli(), colName, q.TimeRange.To.UnixMilli())
		}
	})

	// 2. $__timeFrom()
	reTimeFrom := regexp.MustCompile(`\$__timeFrom\(\)`)
	sql = reTimeFrom.ReplaceAllStringFunc(sql, func(match string) string {
		return kineticaTimeLiteral(q.TimeRange.From)
	})

	// 3. $__timeTo()
	reTimeTo := regexp.MustCompile(`\$__timeTo\(\)`)
	sql = reTimeTo.ReplaceAllStringFunc(sql, func(match string) string {
		return kineticaTimeLiteral(q.TimeRange.To)
	})

	// 4. $__unixEpochFrom()
	reUnixFrom := regexp.MustCompile(`\$__unixEpochFrom\(\)`)
	sql = reUnixFrom.ReplaceAllStringFunc(sql, func(match string) string {
		return fmt.Sprintf("%d", q.TimeRange.From.UnixMilli())
	})

	// 5. $__unixEpochTo()
	reUnixTo := regexp.MustCompile(`\$__unixEpochTo\(\)`)
	sql = reUnixTo.ReplaceAllStringFunc(sql, func(match string) string {
		return fmt.Sprintf("%d", q.TimeRange.To.UnixMilli())
	})

	// 6. $__timeGroup(col, interval)
	// ReplaceAllStringFunc cannot return an error, so the first failure is captured
	// and reported after the walk rather than silently leaving the macro in place.
	reTimeGroup := regexp.MustCompile(`\$__timeGroup\(([^,()]+),([^()]+)\)`)
	var timeGroupErr error
	sql = reTimeGroup.ReplaceAllStringFunc(sql, func(match string) string {
		if timeGroupErr != nil {
			return match
		}
		submatches := reTimeGroup.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		colName := strings.TrimSpace(submatches[1])
		millis, err := parseIntervalMillis(submatches[2])
		if err != nil {
			timeGroupErr = err
			return match
		}
		return timeGroupExpr(colName, millis)
	})
	if timeGroupErr != nil {
		return "", timeGroupErr
	}

	// 7. $__labelMatch(column, 'key', values) and $__labelRegex(column, 'key', patterns)
	sql, err := expandLabelMacros(sql)
	if err != nil {
		return "", err
	}

	return sql, nil
}

// kineticaTimeLayout - how a time is written in a Kinetica timestamp literal
const kineticaTimeLayout = "2006-01-02 15:04:05.000"

// kineticaTimeLiteral - a SQL literal of a time, in UTC. Kinetica reads a
// timestamp literal, which carries no zone, as UTC, while the SDK builds a
// query's time range in the plugin process's local zone.
func kineticaTimeLiteral(t time.Time) string {
	return "'" + t.UTC().Format(kineticaTimeLayout) + "'"
}

func getColumnIsString(ctx context.Context, db *kinetica.Kinetica, schema, table, colName string) (bool, error) {
	cleanColName := strings.Trim(colName, "\"")
	// A column qualified by a table alias (p.ts) is looked up by its own name
	if i := strings.LastIndex(cleanColName, "."); i >= 0 {
		cleanColName = strings.Trim(cleanColName[i+1:], "\"")
	}

	if table == "" {
		return false, nil
	}

	fullTableName := table
	if schema != "" && !strings.HasPrefix(table, schema+".") {
		fullTableName = fmt.Sprintf("%s.%s", schema, table)
	}

	resp, err := db.ShowTableRawWithOpts(ctx, fullTableName, &kinetica.ShowTableOptions{GetColumnInfo: true})
	if err != nil {
		return false, err
	}
	if len(resp.TypeSchemas) == 0 {
		return false, nil
	}

	var typeDef KineticaTypeDefinition
	if err := json.Unmarshal([]byte(resp.TypeSchemas[0]), &typeDef); err != nil {
		return false, err
	}

	for _, f := range typeDef.Fields {
		if f.Name == cleanColName {
			typeStr := strings.ToLower(fmt.Sprintf("%v", f.Type))
			return strings.Contains(typeStr, "string") ||
				strings.Contains(typeStr, "date") ||
				strings.Contains(typeStr, "char"), nil
		}
	}
	return false, nil
}

// -------------------------------------------------------------------------
// 3. RESOURCE HANDLER
// -------------------------------------------------------------------------

func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	parsedUrl, err := url.Parse(req.URL)
	if err != nil {
		logError("Failed to parse request URL", "url", req.URL, "err", err)
		return sendError(sender, fmt.Errorf("invalid request URL"))
	}
	qParams := parsedUrl.Query()

	// SCHEMAS
	if req.Path == "schemas" {
		resp, err := d.client.ShowTableRawWithOpts(ctx, "", &kinetica.ShowTableOptions{ShowChildren: true})
		if err != nil {
			return sendError(sender, err)
		}
		body, _ := json.Marshal(resp.TableNames)
		return sender.Send(&backend.CallResourceResponse{Status: 200, Body: body})
	}

	// TABLES
	if req.Path == "tables" {
		schema := qParams.Get("schema")
		if schema == "" {
			return sendError(sender, fmt.Errorf("missing 'schema' query parameter"))
		}
		resp, err := d.client.ShowTableRawWithOpts(ctx, schema, &kinetica.ShowTableOptions{ShowChildren: true})
		if err != nil {
			return sendError(sender, err)
		}
		tables := make([]string, len(resp.TableNames))
		for i, t := range resp.TableNames {
			tables[i] = fmt.Sprintf("%s.%s", schema, t)
		}
		body, _ := json.Marshal(tables)
		return sender.Send(&backend.CallResourceResponse{Status: 200, Body: body})
	}

	// COLUMNS
	if req.Path == "columns" {
		schema := qParams.Get("schema")
		table := qParams.Get("table")

		fullTableName := table
		if schema != "" && !strings.HasPrefix(table, schema+".") {
			fullTableName = fmt.Sprintf("%s.%s", schema, table)
		}

		resp, err := d.client.ShowTableRawWithOpts(ctx, fullTableName, &kinetica.ShowTableOptions{GetColumnInfo: true})
		if err != nil {
			return sendError(sender, err)
		}

		var colNames []string
		if len(resp.TypeSchemas) != 0 {
			var typeDef KineticaTypeDefinition
			if json.Unmarshal([]byte(resp.TypeSchemas[0]), &typeDef) == nil {
				for _, f := range typeDef.Fields {
					colNames = append(colNames, f.Name)
				}
			}
		}
		body, _ := json.Marshal(colNames)
		return sender.Send(&backend.CallResourceResponse{Status: 200, Body: body})
	}

	return sender.Send(&backend.CallResourceResponse{Status: 404})
}

func sendError(sender backend.CallResourceResponseSender, err error) error {
	logError("Resource request failed", "err", err)
	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusInternalServerError,
		Body:   []byte("Request failed, see Grafana server log for details"),
	})
}

// -------------------------------------------------------------------------
// HELPERS
// -------------------------------------------------------------------------

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if _, err := d.client.ShowSystemPropertiesRaw(ctx); err != nil {
		logError("CheckHealth: connection to Kinetica failed", "err", err)
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Connection to Kinetica failed, see Grafana server log for details"}, nil
	}
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "Success"}, nil
}

func (d *Datasource) Dispose() {}

func loadSettings(s *backend.DataSourceInstanceSettings) (*DataSourceSettings, error) {
	if s == nil {
		return nil, fmt.Errorf("settings is nil")
	}
	var cfg DataSourceSettings
	if err := json.Unmarshal(s.JSONData, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func createClient(ctx context.Context, cfg *DataSourceSettings, s *backend.DataSourceInstanceSettings) *kinetica.Kinetica {
	dbUrl := s.URL
	if dbUrl == "" {
		dbUrl = cfg.URL
	}

	// Default to localhost if URL is missing
	if dbUrl == "" {
		dbUrl = "http://host.docker.internal:9191"
	}

	if dbUrl != "" && !strings.Contains(dbUrl, "://") {
		dbUrl = "http://" + dbUrl
	}

	opts := kinetica.KineticaOptions{
		Username:           cfg.Username,
		ByPassSslCertCheck: cfg.TlsSkipVerify,
	}
	if s.DecryptedSecureJSONData != nil {
		opts.Password = s.DecryptedSecureJSONData["password"]
	}
	return kinetica.NewWithOptions(ctx, dbUrl, &opts)
}

// kineticaTimeLayouts are the string forms Kinetica uses for its date and datetime
// types, most precise first.
var kineticaTimeLayouts = []string{
	"2006-01-02 15:04:05.000",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05.000Z07:00",
	"2006-01-02",
}

// parseKineticaTimes converts a column of Kinetica time strings to timestamps.
// It reports false unless every non-null value parses, so a column that only looks
// like a time is left as text rather than being half-converted.
func parseKineticaTimes(sliceData []any) ([]*time.Time, bool) {
	vector := make([]*time.Time, len(sliceData))
	for i, v := range sliceData {
		if v == nil {
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, false
		}
		parsed := false
		for _, layout := range kineticaTimeLayouts {
			if t, err := time.Parse(layout, str); err == nil {
				vector[i] = &t
				parsed = true
				break
			}
		}
		if !parsed {
			return nil, false
		}
	}
	return vector, true
}

// resultColumn - a column of a query's result
type resultColumn struct {
	name string
	// datatype is Kinetica's name of the column's type, lower case, such as
	// timestamp, datetime or decimal(28,6); "" when the response names none
	datatype string
	values   []any
}

// resultColumns - the columns of a decoded /execute/sql response, in the order
// the query selects them, with the types the response gives them
//
//	@return bool - false when the response has no column headers
func resultColumns(rawMap map[string]any) ([]resultColumn, bool) {
	headersVar, ok := rawMap["column_headers"]
	if !ok {
		return nil, false
	}
	headers, _ := headersVar.([]any)
	datatypes, _ := rawMap["column_datatypes"].([]any)

	columns := make([]resultColumn, 0, len(headers))
	for i, h := range headers {
		values, ok := rawMap[fmt.Sprintf("column_%d", i+1)].([]any)
		if !ok {
			continue
		}
		column := resultColumn{name: fmt.Sprintf("%v", h), values: values}
		if i < len(datatypes) {
			column.datatype = strings.ToLower(fmt.Sprintf("%v", datatypes[i]))
		}
		columns = append(columns, column)
	}
	return columns, true
}

// hasDatatypes - whether the response named the type of every column; older
// servers send no column_datatypes
func hasDatatypes(columns []resultColumn) bool {
	for _, c := range columns {
		if c.datatype == "" {
			return false
		}
	}
	return len(columns) > 0
}

// isTimeDatatype - whether a Kinetica type is a point in time: timestamp
// (milliseconds since the epoch), datetime or date (text)
func isTimeDatatype(datatype string) bool {
	switch datatype {
	case "timestamp", "datetime", "date":
		return true
	}
	return false
}

// isDecimalDatatype - whether a Kinetica type is a decimal, which the
// response carries as text
func isDecimalDatatype(datatype string) bool {
	return strings.HasPrefix(datatype, "decimal")
}

// parseDecimals - a column of decimal text as numbers
//
//	@return bool - false unless every non-null value parses
func parseDecimals(sliceData []any) ([]*float64, bool) {
	vector := make([]*float64, len(sliceData))
	for i, v := range sliceData {
		if v == nil {
			continue
		}
		str, ok := v.(string)
		if !ok {
			return nil, false
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			return nil, false
		}
		vector[i] = &f
	}
	return vector, true
}

// parseToFrame - the frame of a query's result, its fields in the order the
// query selects its columns. A column is a time field when its type is a time
// type, or its name is in timeCols.
func parseToFrame(columns []resultColumn, timeCols map[string]bool, refID string) (*data.Frame, error) {
	frame := data.NewFrame("response")
	frame.RefID = refID

	for _, column := range columns {
		colName := column.name
		sliceData := column.values

		rowCount := len(sliceData)
		if rowCount == 0 {
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, []string{}))
			continue
		}

		isTimeColumn := timeCols[colName] || isTimeDatatype(column.datatype)
		var sampleVal any
		for _, v := range sliceData {
			if v != nil {
				sampleVal = v
				break
			}
		}

		switch sampleVal.(type) {
		case float64, float32:
			vector := make([]*float64, rowCount)
			for i, v := range sliceData {
				if v != nil {
					if f, ok := v.(float64); ok {
						vector[i] = &f
					} else if f32, ok := v.(float32); ok {
						f := float64(f32)
						vector[i] = &f
					}
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))

		case int, int64, int32:
			if isTimeColumn {
				vector := make([]*time.Time, rowCount)
				for i, v := range sliceData {
					if v != nil {
						var val int64
						switch n := v.(type) {
						case int:
							val = int64(n)
						case int32:
							val = int64(n)
						case int64:
							val = n
						case float64:
							val = int64(n)
						}
						t := time.UnixMilli(val)
						vector[i] = &t
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))
			} else {
				vector := make([]*int64, rowCount)
				for i, v := range sliceData {
					if v != nil {
						var val int64
						switch n := v.(type) {
						case int:
							val = int64(n)
						case int32:
							val = int64(n)
						case int64:
							val = n
						case float64:
							val = int64(n)
						}
						vector[i] = &val
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))
			}

		case string:
			// Kinetica returns its string-backed time types (date, datetime, and the
			// datetime that TIME_BUCKET produces) as formatted strings rather than
			// epoch integers, so they need parsing before Grafana will plot them.
			if isTimeColumn {
				if times, ok := parseKineticaTimes(sliceData); ok {
					frame.Fields = append(frame.Fields, data.NewField(colName, nil, times))
					continue
				}
				logWarn("Time column did not parse as a timestamp; leaving it as text", "column", colName)
			}
			if isDecimalDatatype(column.datatype) {
				if numbers, ok := parseDecimals(sliceData); ok {
					frame.Fields = append(frame.Fields, data.NewField(colName, nil, numbers))
					continue
				}
				logWarn("Decimal column did not parse as numbers; leaving it as text", "column", colName)
			}
			vector := make([]*string, rowCount)
			for i, v := range sliceData {
				if v != nil {
					if s, ok := v.(string); ok {
						vector[i] = &s
					}
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))

		case bool:
			vector := make([]*bool, rowCount)
			for i, v := range sliceData {
				if v != nil {
					if b, ok := v.(bool); ok {
						vector[i] = &b
					}
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))

		default:
			vector := make([]*string, rowCount)
			for i, v := range sliceData {
				if v != nil {
					s := fmt.Sprintf("%v", v)
					vector[i] = &s
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, vector))
		}
	}
	return frame, nil
}

// -------------------------------------------------------------------------
// 5. LOGGING HELPERS
// -------------------------------------------------------------------------

func logError(msg string, args ...any) { addCallerContext("error", msg, args...) }
func logWarn(msg string, args ...any)  { addCallerContext("warn", msg, args...) }
func logInfo(msg string, args ...any)  { addCallerContext("info", msg, args...) }
func logDebug(msg string, args ...any) { addCallerContext("debug", msg, args...) }

func addCallerContext(level string, msg string, args ...any) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	funcName := runtime.FuncForPC(pc).Name()
	if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
		funcName = funcName[lastSlash+1:]
	}
	finalArgs := append(args, "source", fmt.Sprintf("%s:%d", filepath.Base(file), line), "func", funcName)

	switch level {
	case "error":
		log.DefaultLogger.Error(msg, finalArgs...)
	case "warn":
		log.DefaultLogger.Warn(msg, finalArgs...)
	case "info":
		log.DefaultLogger.Info(msg, finalArgs...)
	case "debug":
		log.DefaultLogger.Debug(msg, finalArgs...)
	}
}

// toTimeSeriesFrame - a long frame (one row per time and combination of its
// string columns) as a wide one, with a value field per combination, labelled
// by it; any other frame as it is. Rows are sorted by time first, as the
// conversion requires, and rows without a time are left out.
func toTimeSeriesFrame(frame *data.Frame) (*data.Frame, error) {
	schema := frame.TimeSeriesSchema()
	if schema.Type != data.TimeSeriesTypeLong {
		return frame, nil
	}
	rowCount, err := frame.RowLen()
	if err != nil {
		return nil, err
	}

	timeField := frame.Fields[schema.TimeIndex]
	timeAt := func(row int) (time.Time, bool) {
		v, ok := timeField.ConcreteAt(row)
		if !ok {
			return time.Time{}, false
		}
		t, ok := v.(time.Time)
		return t, ok
	}
	rows := make([]int, 0, rowCount)
	for row := 0; row < rowCount; row++ {
		if _, ok := timeAt(row); ok {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(a, b int) bool {
		ta, _ := timeAt(rows[a])
		tb, _ := timeAt(rows[b])
		return ta.Before(tb)
	})

	sorted := frame.EmptyCopy()
	for _, row := range rows {
		sorted.AppendRow(frame.RowCopy(row)...)
	}
	return data.LongToWide(sorted, &data.FillMissing{Mode: data.FillModeNull})
}
