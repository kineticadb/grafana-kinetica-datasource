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
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/hamba/avro/v2"
	"github.com/kineticadb/kinetica-api-go/kinetica"
)

var (
	_ backend.QueryDataHandler    = (*Datasource)(nil)
	_ backend.CheckHealthHandler  = (*Datasource)(nil)
	_ backend.CallResourceHandler = (*Datasource)(nil)
)

type Datasource struct{}

func NewDatasource(ctx context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// -------------------------------------------------------------------------
// 1. QUERY DATA
// -------------------------------------------------------------------------
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	config, err := loadSettings(req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		return nil, err
	}

	db := createClient(config, req.PluginContext.DataSourceInstanceSettings)
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := d.query(ctx, db, q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *Datasource) query(ctx context.Context, db *kinetica.Kinetica, query backend.DataQuery) backend.DataResponse {

	logInfo("Starting query processing", "refId", query.RefID)

	// 1. Parse Query Model
	var qModel QueryModel
	if err := json.Unmarshal(query.JSON, &qModel); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "Failed to unmarshal query model")
	}

	// 2. Prepare SQL (Macros)
	sql, err := prepareSql(ctx, query, db)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Macro error: %v", err))
	}
	if sql == "" {
		return backend.DataResponse{}
	}

	logInfo("Macro replacement complete", "refId", query.RefID, "final_sql", sql)

	// 3. Determine Table Name for Metadata Lookup
	fullTableName := qModel.Builder.Table

	// A. Try Builder
	if qModel.Builder.Schema != "" && !strings.HasPrefix(qModel.Builder.Table, qModel.Builder.Schema+".") {
		fullTableName = fmt.Sprintf("%s.%s", qModel.Builder.Schema, qModel.Builder.Table)
	}

	// B. Fallback: Raw SQL
	if fullTableName == "" || fullTableName == "." {
		// Regex to find "FROM <table_name>"
		// Matches: FROM table, FROM schema.table, FROM "schema"."table"
		re := regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z0-9_."]+)`)
		match := re.FindStringSubmatch(sql)
		if len(match) > 1 {
			fullTableName = match[1]
			// Clean up quotes if present (e.g., "schema"."table" -> schema.table)
			fullTableName = strings.ReplaceAll(fullTableName, "\"", "")
		}
		logDebug("Table name extracted from Raw SQL", "table", fullTableName)
	}

	// 4. Identify Time Columns (Base Table)
	timeCols, err := d.getTimeColumns(ctx, db, fullTableName)
	if err != nil {
		logWarn("Could not fetch table metadata", "table", fullTableName, "err", err)
		timeCols = make(map[string]bool)
	}

	// 5. Handle Aliases & Aggregates
	// We map aliases in the `timeCols` map so `parseToFrame` knows how to handle them.
	if len(qModel.Builder.Selects) > 0 {
		for _, sel := range qModel.Builder.Selects {
			if sel.Alias != "" && sel.Column != "" {
				// A. Clean Quotes sent by frontend
				cleanCol := strings.Trim(sel.Column, "\"")
				cleanAlias := strings.Trim(sel.Alias, "\"")

				// B. Check if original column is a Time column
				if isTime, ok := timeCols[cleanCol]; ok && isTime {

					// C. AGGREGATE SAFETY CHECK
					// Only preserve "Time" type if the function preserves time (MIN, MAX, or None).
					// COUNT, SUM, AVG of a date returns a Number, not a Date.
					agg := strings.ToUpper(sel.Aggregate)
					isTimePreserving := agg == "" || agg == "MIN" || agg == "MAX" || agg == "FIRST" || agg == "LAST"

					if isTimePreserving {
						timeCols[cleanAlias] = true
					}
				}
			}
		}
	}

	// 6. Execute SQL
	logDebug("Sending query to Kinetica", "refId", query.RefID)
	options := &kinetica.ExecuteSqlOptions{Encoding: "binary"}
	rawResp, err := db.ExecuteSqlRawWithOpts(ctx, sql, 0, -9999, "", nil, options)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadGateway, fmt.Sprintf("Kinetica Error: %v", err))
	}

	// 7. Parse Avro Schema
	schema, err := avro.Parse(rawResp.ResponseSchema)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("Schema Parse Error: %v", err))
	}

	// 8. Unmarshal Data
	rawMap := make(map[string]any)
	err = avro.Unmarshal(schema, rawResp.BinaryEncodedResponse, &rawMap)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("Avro Unmarshal Error: %v", err))
	}

	// 9. Extract Headers & Clean Data
	headersVar, ok := rawMap["column_headers"]
	if !ok {
		return backend.DataResponse{Frames: data.Frames{data.NewFrame("response")}}
	}
	headersInterface := headersVar.([]any)

	cleanData := make(map[string]any, len(headersInterface))
	for i, h := range headersInterface {
		headerName := fmt.Sprintf("%v", h)
		internalKey := fmt.Sprintf("column_%d", i+1)

		if colData, exists := rawMap[internalKey]; exists {
			cleanData[headerName] = colData
		}
	}

	// 10. Convert to Frame
	frame, err := parseToFrame(&cleanData, timeCols, query.RefID)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, err.Error())
	}

	return backend.DataResponse{Frames: data.Frames{frame}}
}

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

		isString, err := getColumnIsString(ctx, db, qModel.Builder.Schema, qModel.Builder.Table, colName)
		if err != nil {
			logError("Failed to lookup column type", "err", err)
			isString = false
		}

		if isString {
			fromStr := q.TimeRange.From.Format("2006-01-02 15:04:05.000")
			toStr := q.TimeRange.To.Format("2006-01-02 15:04:05.000")
			return fmt.Sprintf("%s >= '%s' AND %s <= '%s'", colName, fromStr, colName, toStr)
		} else {
			return fmt.Sprintf("%s >= %d AND %s <= %d", colName, q.TimeRange.From.UnixMilli(), colName, q.TimeRange.To.UnixMilli())
		}
	})

	// 2. $__timeFrom()
	reTimeFrom := regexp.MustCompile(`\$__timeFrom\(\)`)
	sql = reTimeFrom.ReplaceAllStringFunc(sql, func(match string) string {
		return fmt.Sprintf("'%s'", q.TimeRange.From.Format("2006-01-02 15:04:05.000"))
	})

	// 3. $__timeTo()
	reTimeTo := regexp.MustCompile(`\$__timeTo\(\)`)
	sql = reTimeTo.ReplaceAllStringFunc(sql, func(match string) string {
		return fmt.Sprintf("'%s'", q.TimeRange.To.Format("2006-01-02 15:04:05.000"))
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

	return sql, nil
}

func getColumnIsString(ctx context.Context, db *kinetica.Kinetica, schema, table, colName string) (bool, error) {
	cleanColName := strings.Trim(colName, "\"")

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
	config, _ := loadSettings(req.PluginContext.DataSourceInstanceSettings)
	db := createClient(config, req.PluginContext.DataSourceInstanceSettings)

	parsedUrl, _ := url.Parse(req.URL)
	qParams := parsedUrl.Query()

	// SCHEMAS
	if req.Path == "schemas" {
		resp, err := db.ShowTableRawWithOpts(ctx, "", &kinetica.ShowTableOptions{ShowChildren: true})
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
		resp, err := db.ShowTableRawWithOpts(ctx, schema, &kinetica.ShowTableOptions{ShowChildren: true})
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

		resp, err := db.ShowTableRawWithOpts(ctx, fullTableName, &kinetica.ShowTableOptions{GetColumnInfo: true})
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
	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusInternalServerError,
		Body:   []byte(err.Error()),
	})
}

// -------------------------------------------------------------------------
// HELPERS
// -------------------------------------------------------------------------

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	config, err := loadSettings(req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: "Config error"}, nil
	}
	db := createClient(config, req.PluginContext.DataSourceInstanceSettings)
	if _, err := db.ShowSystemPropertiesRaw(ctx); err != nil {
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: err.Error()}, nil
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

func createClient(cfg *DataSourceSettings, s *backend.DataSourceInstanceSettings) *kinetica.Kinetica {
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
		ByPassSslCertCheck: true,
	}
	if s.DecryptedSecureJSONData != nil {
		opts.Password = s.DecryptedSecureJSONData["password"]
	}
	return kinetica.NewWithOptions(context.TODO(), dbUrl, &opts)
}

func parseToFrame(results *map[string]any, timeCols map[string]bool, refID string) (*data.Frame, error) {
	frame := data.NewFrame("response")
	frame.RefID = refID

	if results == nil || len(*results) == 0 {
		return frame, nil
	}

	var colNames []string
	for k := range *results {
		colNames = append(colNames, k)
	}
	sort.Strings(colNames)

	for _, colName := range colNames {
		rawColData := (*results)[colName]
		sliceData, ok := rawColData.([]any)
		if !ok {
			continue
		}

		rowCount := len(sliceData)
		if rowCount == 0 {
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, []string{}))
			continue
		}

		isTimeColumn := timeCols[colName]
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
