package plugin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestParseIntervalMillis(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"500ms", 500},
		{"30s", 30 * 1000},
		{"5m", 5 * 60 * 1000},
		{"1h", 60 * 60 * 1000},
		{"7d", 7 * 24 * 60 * 60 * 1000},
		{"2w", 2 * 7 * 24 * 60 * 60 * 1000},
		// Users write the interval quoted as well as bare, and Grafana substitutes
		// $__interval_ms as a plain number of milliseconds.
		{"'5m'", 5 * 60 * 1000},
		{"\"5m\"", 5 * 60 * 1000},
		{"  5m  ", 5 * 60 * 1000},
		{"5M", 5 * 60 * 1000},
		{"60000", 60000},
	}
	for _, c := range cases {
		got, err := parseIntervalMillis(c.in)
		if err != nil {
			t.Errorf("parseIntervalMillis(%q) returned error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseIntervalMillis(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseIntervalMillisRejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "   ", "abc", "5x", "-5m", "0s", "m"} {
		if got, err := parseIntervalMillis(in); err == nil {
			t.Errorf("parseIntervalMillis(%q) = %d, want an error", in, got)
		}
	}
}

func TestTimeGroupExpr(t *testing.T) {
	// TIME_BUCKET takes the width in milliseconds, which is what parseIntervalMillis
	// yields, so no unit conversion is needed.
	got := timeGroupExpr("ts", 5*60*1000)
	want := "TIME_BUCKET(300000, TIMESTAMP(ts))"
	if got != want {
		t.Errorf("timeGroupExpr:\n got %s\nwant %s", got, want)
	}
}

func TestTimeGroupExprSubSecond(t *testing.T) {
	// TIME_BUCKET supports sub-second widths, unlike a UNIX_TIMESTAMP-based bucket.
	if got, want := timeGroupExpr("ts", 500), "TIME_BUCKET(500, TIMESTAMP(ts))"; got != want {
		t.Errorf("timeGroupExpr = %s, want %s", got, want)
	}
}

func TestTimeGroupExprWrapsColumnInTimestamp(t *testing.T) {
	// Without the TIMESTAMP() wrap, an epoch-milliseconds column is read as a
	// time-of-day and the date is silently dropped.
	if got := timeGroupExpr("epoch_ms", 60000); got != "TIME_BUCKET(60000, TIMESTAMP(epoch_ms))" {
		t.Errorf("expected the column to be wrapped in TIMESTAMP(), got %s", got)
	}
}

func TestKineticaTimeLiteralIsUTC(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}
	// The SDK builds a query's time range with time.Unix, in the process's
	// local zone; Kinetica reads a timestamp literal as UTC
	saved := time.Local
	time.Local = newYork
	defer func() { time.Local = saved }()

	from := time.Unix(0, int64(1791813600000)*int64(time.Millisecond)) // 2026-10-12 14:00:00 UTC
	if got, want := kineticaTimeLiteral(from), "'2026-10-12 14:00:00.000'"; got != want {
		t.Errorf("kineticaTimeLiteral = %s, want %s", got, want)
	}
}

func TestFromTableName(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM otel.metric_series WHERE x = 1":  "otel.metric_series",
		`SELECT * FROM "otel"."metric_series"`:          "otel.metric_series",
		"select a from t1 join t2 on t1.id = t2.id":     "t1",
		"SELECT b FROM (SELECT b FROM s.inner_table) x": "s.inner_table",
		"SELECT 1": "",
		"SELECT x FROM UNNEST_JSON_ARRAY(SELECT a FROM s.t) WITH ORDINALITY": "UNNEST_JSON_ARRAY",
	}
	for sql, want := range cases {
		if got := fromTableName(sql); got != want {
			t.Errorf("fromTableName(%q) = %q, want %q", sql, got, want)
		}
	}
}

func TestPrepareSqlTimeMacrosAreUTC(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}
	saved := time.Local
	time.Local = newYork
	defer func() { time.Local = saved }()

	q := backend.DataQuery{
		JSON: json.RawMessage(`{"rawSql": "SELECT * FROM t WHERE ts BETWEEN $__timeFrom() AND $__timeTo()"}`),
		TimeRange: backend.TimeRange{
			From: time.Unix(0, int64(1791813600000)*int64(time.Millisecond)), // 14:00 UTC, 10:00 in New York
			To:   time.Unix(0, int64(1791817200000)*int64(time.Millisecond)), // 15:00 UTC
		},
	}
	sql, err := prepareSql(context.Background(), q, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM t WHERE ts BETWEEN '2026-10-12 14:00:00.000' AND '2026-10-12 15:00:00.000'"
	if sql != want {
		t.Errorf("prepareSql:\n got %s\nwant %s", sql, want)
	}
}
