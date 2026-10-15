package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/kineticadb/kinetica-api-go/kinetica"
)

// TestLiveQueries runs queries through the datasource against a live
// Kinetica server: column order and types, time detection from the result's
// column types, UTC time macros, the time series format, the label macros and
// $__timeFilter on raw SQL. It needs a server with enable_text_search = true,
// and is skipped unless KINETICA_TEST_URL is set (with KINETICA_TEST_USER and
// KINETICA_TEST_PASSWORD). It works in a schema of its own, dropped at the end.
func TestLiveQueries(t *testing.T) {
	url := os.Getenv("KINETICA_TEST_URL")
	if url == "" {
		t.Skip("set KINETICA_TEST_URL, KINETICA_TEST_USER and KINETICA_TEST_PASSWORD to run against a server")
	}
	ctx := context.Background()
	settings := backend.DataSourceInstanceSettings{
		URL:                     url,
		JSONData:                json.RawMessage(fmt.Sprintf(`{"username": %q}`, os.Getenv("KINETICA_TEST_USER"))),
		DecryptedSecureJSONData: map[string]string{"password": os.Getenv("KINETICA_TEST_PASSWORD")},
	}
	instance, err := NewDatasource(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	ds := instance.(*Datasource)

	schema := fmt.Sprintf("grafana_it_%d", time.Now().UnixNano())
	exec := func(t *testing.T, sql string) {
		t.Helper()
		if _, err := ds.client.ExecuteSqlRawWithOpts(ctx, sql, 0, -9999, "", nil, &kinetica.ExecuteSqlOptions{Encoding: "json"}); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	// Registered first, so that a failure part way through still drops the schema
	t.Cleanup(func() {
		_, _ = ds.client.ExecuteSqlRawWithOpts(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE", 0, -9999, "", nil,
			&kinetica.ExecuteSqlOptions{Encoding: "json"})
	})
	exec(t, "CREATE SCHEMA "+schema)
	table := schema + ".m"
	exec(t, `CREATE TABLE `+table+` (
		ts TIMESTAMP, svc VARCHAR(32), value DOUBLE, ratio DECIMAL(10,4), d DATE, dts VARCHAR(32),
		labels JSON(text_search)
	) TEXT INDEX (labels) WITH OPTIONS (analyzer = 'keyword')`)
	// Two services a minute apart for three minutes, and three rows whose
	// labels need escaping
	exec(t, `INSERT INTO `+table+` VALUES
		('2026-10-12 10:00:00', 'cart', 1, 0.25, '2026-10-12', '2026-10-12 10:00:00', '{"service.name":"cart","http.route":"/api/v1/cart"}'),
		('2026-10-12 10:01:00', 'cart', 2, 0.25, '2026-10-12', '2026-10-12 10:01:00', '{"service.name":"cart","http.route":"/api/v1/cart"}'),
		('2026-10-12 10:02:00', 'cart', 3, 0.25, '2026-10-12', '2026-10-12 10:02:00', '{"service.name":"cart","http.route":"/api/v1/cart"}'),
		('2026-10-12 10:00:00', 'pay', 10, 0.5, '2026-10-12', '2026-10-12 10:00:00', '{"service.name":"pay","http.route":"/api/v1/pay"}'),
		('2026-10-12 10:01:00', 'pay', 20, 0.5, '2026-10-12', '2026-10-12 10:01:00', '{"service.name":"pay","http.route":"/api/v1/pay"}'),
		('2026-10-12 10:02:00', 'pay', 30, 0.5, '2026-10-12', '2026-10-12 10:02:00', '{"service.name":"pay","http.route":"/api/v1/pay"}'),
		('2026-10-11 09:00:00', 'odd', 0, 0, '2026-10-11', '2026-10-11 09:00:00', '{"service.name":"say \"hi\""}'),
		('2026-10-11 09:00:00', 'odd', 0, 0, '2026-10-11', '2026-10-11 09:00:00', '{"service.name":"o''brien"}'),
		('2026-10-11 09:00:00', 'odd', 0, 0, '2026-10-11', '2026-10-11 09:00:00', '{"service.name":"back\\slash"}')`)

	base := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	wholeDay := [2]time.Time{base.Add(-10 * time.Hour), base.Add(10 * time.Hour)}

	run := func(t *testing.T, rawSql, format string, timeRange [2]time.Time) *data.Frame {
		t.Helper()
		model, _ := json.Marshal(map[string]string{"rawSql": strings.ReplaceAll(rawSql, "$table", table), "format": format})
		resp := ds.query(ctx, ds.client, backend.DataQuery{
			RefID:     "A",
			JSON:      model,
			TimeRange: backend.TimeRange{From: timeRange[0], To: timeRange[1]},
		})
		if resp.Error != nil {
			t.Fatalf("%s: %v", rawSql, resp.Error)
		}
		return resp.Frames[0]
	}
	count := func(t *testing.T, where string, timeRange [2]time.Time) int64 {
		t.Helper()
		frame := run(t, "SELECT COUNT(*) AS n FROM $table m WHERE "+where, "", timeRange)
		v, _ := frame.Fields[0].ConcreteAt(0)
		return v.(int64)
	}

	t.Run("columns in the query's order, time from the column types", func(t *testing.T) {
		// A time bucket under an alias the plugin cannot know, in a subquery
		frame := run(t, `SELECT svc, b AS when_it_was, at, value, ratio
			FROM (SELECT svc, TIME_BUCKET(60000, ts) AS b, ts AS at, value, ratio FROM $table WHERE svc = 'cart') x
			ORDER BY when_it_was`, "", wholeDay)

		if got := strings.Join(fieldNames(frame), ","); got != "svc,when_it_was,at,value,ratio" {
			t.Errorf("fields = %s", got)
		}
		for _, name := range []string{"when_it_was", "at"} {
			field, _ := frame.FieldByName(name)
			if field.Type() != data.FieldTypeNullableTime {
				t.Errorf("%s is %s, want time", name, field.Type())
			}
		}
		when, _ := frame.FieldByName("when_it_was")
		if v, _ := when.ConcreteAt(1); !v.(time.Time).Equal(base.Add(time.Minute)) {
			t.Errorf("second bucket = %v, want %v", v, base.Add(time.Minute))
		}
		ratio, _ := frame.FieldByName("ratio")
		if ratio.Type() != data.FieldTypeNullableFloat64 {
			t.Errorf("a DECIMAL column is %s, want float64", ratio.Type())
		}
	})

	t.Run("time macros are UTC on a server in another zone", func(t *testing.T) {
		newYork, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Skipf("no time zone data: %v", err)
		}
		saved := time.Local
		time.Local = newYork
		defer func() { time.Local = saved }()

		// The minute from 10:01 UTC, as the SDK builds it: in the local zone
		oneMinute := [2]time.Time{
			time.Unix(0, base.Add(time.Minute).UnixNano()),
			time.Unix(0, base.Add(time.Minute+59*time.Second).UnixNano()),
		}
		if n := count(t, "ts BETWEEN $__timeFrom() AND $__timeTo()", oneMinute); n != 2 {
			t.Errorf("rows in the minute = %d, want 2", n)
		}
	})

	t.Run("time series format", func(t *testing.T) {
		// Rows ordered by service, not time
		frame := run(t, "SELECT ts AS t, svc, value FROM $table WHERE svc <> 'odd' ORDER BY svc, ts DESC", formatTimeSeries, wholeDay)

		if frame.TimeSeriesSchema().Type != data.TimeSeriesTypeWide {
			t.Fatalf("result is %s, want a wide time series", frame.TimeSeriesSchema().Type)
		}
		if n, _ := frame.RowLen(); n != 3 {
			t.Errorf("rows = %d, want 3", n)
		}
		want := map[string][]float64{"cart": {1, 2, 3}, "pay": {10, 20, 30}}
		for _, field := range frame.Fields[1:] {
			var got []float64
			for i := 0; i < field.Len(); i++ {
				v, _ := field.ConcreteAt(i)
				got = append(got, v.(float64))
			}
			if w := want[field.Labels["svc"]]; fmt.Sprint(got) != fmt.Sprint(w) {
				t.Errorf("series %v = %v, want %v", field.Labels, got, w)
			}
		}
	})

	t.Run("label macros", func(t *testing.T) {
		for _, c := range []struct {
			where string
			want  int64
		}{
			{`$__labelMatch(labels, 'service.name', ["cart"])`, 3},
			{`$__labelMatch(m.labels, 'service.name', ["cart", "pay"])`, 6},
			{`$__labelMatch(labels, 'service.name', "say \"hi\"")`, 1},
			{`$__labelMatch(labels, 'service.name', ["o'brien"])`, 1},
			{`$__labelMatch(labels, 'service.name', ["back\\slash"])`, 1},
			{`$__labelMatch(labels, 'service.name', ["Cart"])`, 0},
			{`$__labelMatch(labels, 'service.name', *)`, 9},
			{`$__labelRegex(labels, 'http.route', "/api/v1/.*")`, 6},
			{`$__labelRegex(labels, 'service.name', ["c.*", "p.y"])`, 6},
			{`$__labelRegex(labels, 'http.route', ["/api/v1/cart"]) AND $__labelMatch(labels, 'service.name', "cart")`, 3},
		} {
			if n := count(t, c.where, wholeDay); n != c.want {
				t.Errorf("%s: %d rows, want %d", c.where, n, c.want)
			}
		}
	})

	t.Run("$__timeFilter on raw SQL", func(t *testing.T) {
		// A DATE column cannot be compared with epoch milliseconds, and a text
		// one compares as text, so both need the string form
		for _, column := range []string{"ts", "d", "m.d", "dts"} {
			if n := count(t, "$__timeFilter("+column+")", wholeDay); n != 6 {
				t.Errorf("$__timeFilter(%s): %d rows, want 6", column, n)
			}
		}
	})
}
