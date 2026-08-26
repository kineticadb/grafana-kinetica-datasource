package plugin

import "testing"

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
