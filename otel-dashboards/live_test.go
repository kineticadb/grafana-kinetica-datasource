package oteldashboards

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/kinetica/datasource/pkg/plugin"
	"github.com/kineticadb/kinetica-api-go/kinetica"
)

// base - the time of the first samples of testdata/metrics.sql
var base = time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)

// at - the time some minutes after base
func at(minutes int) time.Time {
	return base.Add(time.Duration(minutes) * time.Minute)
}

// liveDashboards - the plugin's backend, and the schema it reads the metrics
// of testdata/metrics.sql from
type liveDashboards struct {
	ds     *plugin.Datasource
	schema string
}

// startLive - the metrics of testdata/metrics.sql, loaded into a schema of
// their own on the server KINETICA_TEST_URL names (with KINETICA_TEST_USER and
// KINETICA_TEST_PASSWORD), which is dropped when the test ends unless
// KINETICA_TEST_KEEP_SCHEMA is set; the test is skipped when KINETICA_TEST_URL
// is not set
func startLive(t *testing.T) *liveDashboards {
	t.Helper()
	url := os.Getenv("KINETICA_TEST_URL")
	if url == "" {
		t.Skip("set KINETICA_TEST_URL, KINETICA_TEST_USER and KINETICA_TEST_PASSWORD to run against a server")
	}
	ctx := context.Background()
	user, password := os.Getenv("KINETICA_TEST_USER"), os.Getenv("KINETICA_TEST_PASSWORD")
	instance, err := plugin.NewDatasource(ctx, backend.DataSourceInstanceSettings{
		URL:                     url,
		JSONData:                json.RawMessage(fmt.Sprintf(`{"username": %q}`, user)),
		DecryptedSecureJSONData: map[string]string{"password": password},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := kinetica.NewWithOptions(ctx, url, &kinetica.KineticaOptions{Username: user, Password: password})
	execute := func(sql string) error {
		_, err := client.ExecuteSqlRawWithOpts(ctx, sql, 0, -9999, "", nil, &kinetica.ExecuteSqlOptions{Encoding: "json"})
		return err
	}
	schema := fmt.Sprintf("otel_dashboards_%d", time.Now().UnixNano())
	// Registered first, so that a failure part way through still drops the schema
	t.Cleanup(func() {
		if os.Getenv("KINETICA_TEST_KEEP_SCHEMA") != "" {
			// To look at the metrics through the dashboards
			t.Logf("kept schema %s", schema)
			return
		}
		_ = execute("DROP SCHEMA IF EXISTS " + schema + " CASCADE")
	})
	if err := execute("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	for _, statement := range fixtureStatements(t, schema) {
		if err := execute(statement); err != nil {
			t.Fatalf("%v\n%s", err, statement)
		}
	}
	return &liveDashboards{ds: instance.(*plugin.Datasource), schema: schema}
}

// fixtureStatements - the statements of testdata/metrics.sql, for a schema
func fixtureStatements(t *testing.T, schema string) []string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("testdata", "metrics.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(string(text), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			lines = append(lines, line)
		}
	}
	var statements []string
	for _, statement := range strings.Split(strings.Join(lines, "\n"), ";\n") {
		if statement = strings.TrimSpace(statement); statement != "" {
			statements = append(statements, strings.ReplaceAll(statement, "{{schema}}", schema))
		}
	}
	return statements
}

// query - the frame the plugin returns for a dashboard query, interpolated as
// Grafana interpolates it, over the hour either side of base
func (l *liveDashboards) query(t *testing.T, q dashboardQuery, vars grafanaVariables) *data.Frame {
	t.Helper()
	vars.schema = l.schema
	sql := vars.interpolate(t, q.sql)
	model, err := json.Marshal(map[string]string{"rawSql": sql, "format": q.format})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := l.ds.QueryData(context.Background(), &backend.QueryDataRequest{Queries: []backend.DataQuery{{
		RefID:         "A",
		JSON:          model,
		Interval:      time.Minute,
		MaxDataPoints: 1000,
		TimeRange:     backend.TimeRange{From: at(-60), To: at(60)},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	r := resp.Responses["A"]
	if r.Error != nil {
		t.Fatalf("%v\n%s", r.Error, sql)
	}
	if len(r.Frames) != 1 {
		t.Fatalf("%d frames, want 1", len(r.Frames))
	}
	return r.Frames[0]
}

// rows - the rows of a dashboard query's frame, each a map from column name to
// value (nil for a null). The time series a query in the time_series format
// returns are turned back into rows: one per time at which a series has a
// value, holding the time, the series' labels and the value.
func rows(t *testing.T, q dashboardQuery, frame *data.Frame) []map[string]any {
	t.Helper()
	n, err := frame.RowLen()
	if err != nil {
		t.Fatal(err)
	}
	var result []map[string]any
	if q.format != "time_series" {
		for i := 0; i < n; i++ {
			row := map[string]any{}
			for _, field := range frame.Fields {
				row[field.Name], _ = field.ConcreteAt(i)
			}
			result = append(result, row)
		}
		return result
	}

	if n == 0 {
		return nil
	}
	schema := frame.TimeSeriesSchema()
	if schema.Type != data.TimeSeriesTypeWide {
		t.Fatalf("the plugin returned a frame of type %v, want a wide time series", schema.Type)
	}
	timeField := frame.Fields[schema.TimeIndex]
	for f, field := range frame.Fields {
		if f == schema.TimeIndex {
			continue
		}
		for i := 0; i < n; i++ {
			value, ok := field.ConcreteAt(i)
			if !ok {
				continue
			}
			when, _ := timeField.ConcreteAt(i)
			row := map[string]any{timeField.Name: when, field.Name: value}
			for label, labelValue := range field.Labels {
				row[label] = labelValue
			}
			result = append(result, row)
		}
	}
	return result
}

// assertRows - checks that rows are exactly the expected ones, in any order:
// each expected row gives the values of some columns, compared as numbers (to
// 1e-9), as times or as they are
func assertRows(t *testing.T, rows []map[string]any, expected ...map[string]any) {
	t.Helper()
	if len(rows) != len(expected) {
		t.Fatalf("%d rows, want %d: %v", len(rows), len(expected), rows)
	}
	used := make([]bool, len(rows))
next:
	for _, want := range expected {
		for i, row := range rows {
			if !used[i] && rowMatches(t, row, want) {
				used[i] = true
				continue next
			}
		}
		t.Errorf("no row %v in %v", want, rows)
	}
}

func rowMatches(t *testing.T, row map[string]any, want map[string]any) bool {
	t.Helper()
	for column, expected := range want {
		actual, ok := row[column]
		if !ok {
			t.Fatalf("no column %s in %v", column, row)
		}
		switch e := expected.(type) {
		case float64:
			// A number a panel plots or formats must reach it as one
			if a, isNumber := number(actual); !isNumber || math.Abs(a-e) > 1e-9 {
				return false
			}
		case time.Time:
			if a, isTime := actual.(time.Time); !isTime || !a.Equal(e) {
				return false
			}
		default:
			if actual != expected {
				return false
			}
		}
	}
	return true
}

// number - a numeric value as a float64
func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int16:
		return float64(v), true
	case int8:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	}
	return 0, false
}

// TestDashboardsLive runs every query of the dashboards through the plugin's
// backend, as Grafana sends it, against the metrics of testdata/metrics.sql,
// and checks the values each returns: every panel with its variables at All,
// some with filters, and every query variable. It needs a server with
// enable_text_search = true, and is skipped unless KINETICA_TEST_URL is set.
func TestDashboardsLive(t *testing.T) {
	live := startLive(t)
	queries := map[string]dashboardQuery{}
	for _, q := range dashboardQueries(t) {
		queries[q.name()] = q
	}
	run := func(t *testing.T, name string, values map[string][]string) []map[string]any {
		t.Helper()
		q, ok := queries[name]
		if !ok {
			t.Fatalf("no query %s", name)
		}
		return rows(t, q, live.query(t, q, grafanaVariables{values: values}))
	}

	t.Run("panels", func(t *testing.T) {
		expected := map[string][]map[string]any{
			"otel-metrics-http.json/Requests per second": {
				{"time": at(1), "service": "checkout", "requests_per_second": 1.5}},
			"otel-metrics-http.json/5xx responses per second": {
				{"time": at(1), "service": "checkout", "errors_per_second": 10.0 / 60}},
			"otel-metrics-http.json/Error ratio": {
				{"time": at(1), "service": "checkout", "error_ratio": 1.0 / 9}},
			"otel-metrics-http.json/Mean request duration": {
				{"time": at(1), "service": "checkout", "mean_duration": 13.0 / 90}},
			"otel-metrics-http.json/Active requests": {
				{"time": at(0), "service": "checkout", "active_requests": 4.0},
				{"time": at(0), "service": "checkout-v2", "active_requests": 1.0},
				{"time": at(0), "service": "Checkout", "active_requests": 1.0},
				{"time": at(1), "service": "checkout", "active_requests": 6.0},
			},
			"otel-metrics-http.json/Busiest routes": {
				{"service": "checkout", "route": "/api/v1/items/{id}", "requests": 60.0, "error_ratio": 0.0,
					"mean_duration": 0.1},
				{"service": "checkout", "route": "/api/v1/cart", "requests": 30.0, "error_ratio": 1.0 / 3,
					"mean_duration": 7.0 / 30},
			},
			"otel-metrics-http.json/Slowest exemplars": {
				{"point_time_unix": at(1), "duration": 0.9, "service": "checkout", "route": "/api/v1/cart",
					"trace_id": "ab000000000000000000000000000001", "span_id": "cd00000000000001"}},
			"otel-metrics-hosts.json/CPU utilization": {
				{"time": at(1), "host": "h1", "cpu_utilization": 0.25}},
			"otel-metrics-hosts.json/Load average (1m)": {
				{"time": at(0), "host": "h1", "load_average": 1.5},
				{"time": at(1), "host": "h1", "load_average": 2.5},
				{"time": at(0), "host": "h2", "load_average": 0.5},
				{"time": at(1), "host": "h2", "load_average": 0.7},
			},
			"otel-metrics-hosts.json/Memory used": {
				{"time": at(0), "host": "h1", "used_bytes": 1000.0}},
			"otel-metrics-hosts.json/Filesystem utilization": {
				{"time": at(0), "host": "h1", "mountpoint": "/", "utilization": 0.3}},
			"otel-metrics-hosts.json/Network throughput": {
				{"time": at(1), "host": "h1", "direction": "receive", "bytes_per_second": 100.0}},
			"otel-metrics-hosts.json/Disk throughput": {
				{"time": at(1), "host": "h1", "direction": "write", "bytes_per_second": 200.0}},
			"otel-metrics-inventory.json/Metrics": {
				{"metric_name": "http.server.request.duration", "metric_type": "histogram", "unit": "s",
					"series": 3.0, "services": 1.0, "first_seen": at(0)},
				{"metric_name": "http.server.active_requests", "metric_type": "sum", "unit": "{request}",
					"series": 3.0, "services": 3.0},
				{"metric_name": "system.cpu.time", "metric_type": "sum", "unit": "s", "series": 2.0, "services": 0.0},
				{"metric_name": "system.cpu.load_average.1m", "metric_type": "gauge", "unit": "{thread}",
					"series": 2.0, "services": 0.0},
				{"metric_name": "system.memory.usage", "series": 2.0},
				{"metric_name": "system.filesystem.usage", "series": 2.0},
				{"metric_name": "system.network.io", "series": 1.0},
				{"metric_name": "system.disk.io", "series": 1.0},
			},
			"otel-metrics-inventory.json/Series per service": {
				{"service": "checkout", "series": 4.0, "metric_names": 2.0},
				{"service": "checkout-v2", "series": 1.0, "metric_names": 1.0},
				{"service": "Checkout", "series": 1.0, "metric_names": 1.0},
				{"service": "(none)", "series": 10.0, "metric_names": 6.0},
			},
			"otel-metrics-inventory.json/Label keys": {
				{"label": "host.name", "series": 14.0},
				{"label": "service.name", "series": 6.0},
				{"label": "state", "series": 6.0},
				{"label": "http.route", "series": 3.0},
				{"label": "http.response.status_code", "series": 3.0},
				{"label": "cpu", "series": 2.0},
				{"label": "mountpoint", "series": 2.0},
				{"label": "device", "series": 2.0},
				{"label": "direction", "series": 2.0},
			},
			"otel-metrics-inventory.json/Points written": {
				{"time": at(0), "point_table": "number", "points": 13.0},
				{"time": at(1), "point_table": "number", "points": 7.0},
				{"time": at(0), "point_table": "histogram", "points": 3.0},
				{"time": at(1), "point_table": "histogram", "points": 3.0},
			},
			"otel-metrics-inventory.json/New series per hour": {
				{"time": at(0), "metric_type": "histogram", "new_series": 3.0},
				{"time": at(0), "metric_type": "sum", "new_series": 11.0},
				{"time": at(0), "metric_type": "gauge", "new_series": 2.0},
			},
		}

		for _, q := range dashboardQueries(t) {
			if q.panel == nil {
				continue
			}
			t.Run(q.name(), func(t *testing.T) {
				want, ok := expected[q.name()]
				if !ok {
					t.Fatalf("no expected rows for %s", q.name())
				}
				assertRows(t, run(t, q.name(), nil), want...)
			})
		}
	})

	t.Run("filters", func(t *testing.T) {
		for _, c := range []struct {
			name   string
			query  string
			values map[string][]string
			want   []map[string]any
		}{
			{"one service", "otel-metrics-http.json/Active requests", map[string][]string{"service": {"checkout-v2"}},
				[]map[string]any{{"time": at(0), "service": "checkout-v2", "active_requests": 1.0}}},
			{"service names are case-sensitive", "otel-metrics-http.json/Requests per second",
				map[string][]string{"service": {"Checkout"}}, nil},
			{"one route", "otel-metrics-http.json/Requests per second", map[string][]string{"route": {"/api/v1/cart"}},
				[]map[string]any{{"time": at(1), "service": "checkout", "requests_per_second": 0.5}}},
			{"a route with a parameter", "otel-metrics-http.json/Requests per second",
				map[string][]string{"route": {"/api/v1/items/{id}"}},
				[]map[string]any{{"time": at(1), "service": "checkout", "requests_per_second": 1.0}}},
			{"two routes", "otel-metrics-http.json/Busiest routes",
				map[string][]string{"route": {"/api/v1/cart", "/api/v1/items/{id}"}},
				[]map[string]any{{"route": "/api/v1/cart", "requests": 30.0}, {"route": "/api/v1/items/{id}", "requests": 60.0}}},
			{"a route nobody serves", "otel-metrics-http.json/Requests per second",
				map[string][]string{"route": {"/nowhere"}}, nil},
			{"a service and a route", "otel-metrics-http.json/Mean request duration",
				map[string][]string{"service": {"checkout"}, "route": {"/api/v1/cart"}},
				[]map[string]any{{"time": at(1), "service": "checkout", "mean_duration": 7.0 / 30}}},
			{"one host", "otel-metrics-hosts.json/Load average (1m)", map[string][]string{"host": {"h2"}},
				[]map[string]any{{"time": at(0), "host": "h2", "load_average": 0.5},
					{"time": at(1), "host": "h2", "load_average": 0.7}}},
			{"a host without the metric", "otel-metrics-hosts.json/CPU utilization",
				map[string][]string{"host": {"h2"}}, nil},
			{"two services", "otel-metrics-inventory.json/Series per service",
				map[string][]string{"service": {"checkout-v2", "Checkout"}},
				[]map[string]any{{"service": "checkout-v2", "series": 1.0}, {"service": "Checkout", "series": 1.0}}},
			{"one service's metrics", "otel-metrics-inventory.json/Metrics", map[string][]string{"service": {"checkout"}},
				[]map[string]any{{"metric_name": "http.server.request.duration", "series": 3.0},
					{"metric_name": "http.server.active_requests", "series": 1.0}}},
		} {
			t.Run(c.name, func(t *testing.T) {
				assertRows(t, run(t, c.query, c.values), c.want...)
			})
		}
	})

	t.Run("variables", func(t *testing.T) {
		tested := map[string]bool{}
		for _, c := range []struct {
			name   string
			query  string
			values map[string][]string
			want   []string
		}{
			{"services with HTTP metrics", "otel-metrics-http.json/variable service", nil, []string{"checkout"}},
			{"routes", "otel-metrics-http.json/variable route", nil, []string{"/api/v1/cart", "/api/v1/items/{id}"}},
			{"routes of a service", "otel-metrics-http.json/variable route", map[string][]string{"service": {"checkout"}},
				[]string{"/api/v1/cart", "/api/v1/items/{id}"}},
			{"routes of a service without HTTP metrics", "otel-metrics-http.json/variable route",
				map[string][]string{"service": {"Checkout"}}, nil},
			{"hosts", "otel-metrics-hosts.json/variable host", nil, []string{"h1", "h2"}},
			{"services", "otel-metrics-inventory.json/variable service", nil, []string{"Checkout", "checkout", "checkout-v2"}},
		} {
			tested[c.query] = true
			t.Run(c.name, func(t *testing.T) {
				q, ok := queries[c.query]
				if !ok {
					t.Fatalf("no query %s", c.query)
				}
				// A variable takes the first column's values
				frame := live.query(t, q, grafanaVariables{values: c.values})
				var values []string
				if len(frame.Fields) > 0 {
					for i := 0; i < frame.Fields[0].Len(); i++ {
						value, _ := frame.Fields[0].ConcreteAt(i)
						values = append(values, fmt.Sprint(value))
					}
				}
				if !slices.Equal(values, c.want) {
					t.Errorf("values %q, want %q", values, c.want)
				}
			})
		}
		for name, q := range queries {
			if q.panel == nil && !tested[name] {
				t.Errorf("no test of %s", name)
			}
		}
	})
}
