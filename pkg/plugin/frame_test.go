package plugin

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// rawResponse - a decoded /execute/sql response with the given columns
func rawResponse(headers []string, datatypes []string, columns ...[]any) map[string]any {
	raw := map[string]any{}
	h := make([]any, len(headers))
	for i, name := range headers {
		h[i] = name
		raw["column_"+string(rune('1'+i))] = columns[i]
	}
	raw["column_headers"] = h
	if datatypes != nil {
		d := make([]any, len(datatypes))
		for i, t := range datatypes {
			d[i] = t
		}
		raw["column_datatypes"] = d
	}
	return raw
}

func fieldNames(frame *data.Frame) []string {
	names := make([]string, len(frame.Fields))
	for i, f := range frame.Fields {
		names[i] = f.Name
	}
	return names
}

func TestResultColumnsKeepTheQueryOrder(t *testing.T) {
	raw := rawResponse([]string{"zeta", "alpha", "mid"}, []string{"Long", "VARCHAR(16)", "double"},
		[]any{int64(1)}, []any{"a"}, []any{1.5})

	columns, ok := resultColumns(raw)
	if !ok {
		t.Fatal("resultColumns found no headers")
	}
	if len(columns) != 3 || columns[0].name != "zeta" || columns[1].name != "alpha" || columns[2].name != "mid" {
		t.Fatalf("columns = %+v, want zeta, alpha, mid", columns)
	}
	if columns[0].datatype != "long" || columns[1].datatype != "varchar(16)" {
		t.Errorf("datatypes = %q, %q, want lower case", columns[0].datatype, columns[1].datatype)
	}
	if !hasDatatypes(columns) {
		t.Error("hasDatatypes = false, want true")
	}
}

func TestResultColumnsWithoutTypes(t *testing.T) {
	columns, ok := resultColumns(rawResponse([]string{"a"}, nil, []any{int64(1)}))
	if !ok || len(columns) != 1 || columns[0].datatype != "" {
		t.Fatalf("columns = %+v", columns)
	}
	if hasDatatypes(columns) {
		t.Error("hasDatatypes = true for a response without column_datatypes")
	}
	if _, ok := resultColumns(map[string]any{}); ok {
		t.Error("resultColumns of a response without headers reported ok")
	}
}

func TestParseToFrameKeepsTheQueryOrder(t *testing.T) {
	columns, _ := resultColumns(rawResponse([]string{"service", "requests", "error_ratio"}, []string{"char32", "long", "double"},
		[]any{"checkout"}, []any{int64(30)}, []any{0.25}))

	frame, err := parseToFrame(columns, nil, "A")
	if err != nil {
		t.Fatal(err)
	}
	got := fieldNames(frame)
	if len(got) != 3 || got[0] != "service" || got[1] != "requests" || got[2] != "error_ratio" {
		t.Errorf("fields = %v, want service, requests, error_ratio", got)
	}
}

func TestParseToFrameTimeFromTheColumnTypes(t *testing.T) {
	// A timestamp under any alias, a datetime (TIME_BUCKET's result) and a long
	// that only looks like milliseconds
	columns, _ := resultColumns(rawResponse(
		[]string{"anything", "bucket", "plain_ms"},
		[]string{"timestamp", "datetime", "long"},
		[]any{int64(1791799200000)},
		[]any{"2026-10-12 10:01:00.000"},
		[]any{int64(1791799200000)},
	))

	frame, err := parseToFrame(columns, map[string]bool{}, "A")
	if err != nil {
		t.Fatal(err)
	}
	for i, wantTime := range []bool{true, true, false} {
		isTime := frame.Fields[i].Type() == data.FieldTypeNullableTime
		if isTime != wantTime {
			t.Errorf("field %s: time = %v, want %v", frame.Fields[i].Name, isTime, wantTime)
		}
	}
	got, _ := frame.Fields[0].ConcreteAt(0)
	if want := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC); !got.(time.Time).Equal(want) {
		t.Errorf("timestamp = %v, want %v", got, want)
	}
	got, _ = frame.Fields[1].ConcreteAt(0)
	if want := time.Date(2026, 10, 12, 10, 1, 0, 0, time.UTC); !got.(time.Time).Equal(want) {
		t.Errorf("datetime = %v, want %v", got, want)
	}
}

func TestParseToFrameDecimalsAreNumbers(t *testing.T) {
	columns, _ := resultColumns(rawResponse([]string{"ratio", "label"}, []string{"decimal(28,6)", "char16"},
		[]any{"0.333333", nil}, []any{"12.50", "x"}))

	frame, err := parseToFrame(columns, nil, "A")
	if err != nil {
		t.Fatal(err)
	}
	if frame.Fields[0].Type() != data.FieldTypeNullableFloat64 {
		t.Fatalf("decimal field type = %s, want nullable float64", frame.Fields[0].Type())
	}
	if v, _ := frame.Fields[0].ConcreteAt(0); v.(float64) != 0.333333 {
		t.Errorf("decimal = %v", v)
	}
	if _, ok := frame.Fields[0].ConcreteAt(1); ok {
		t.Error("a null decimal is not null")
	}
	if frame.Fields[1].Type() != data.FieldTypeNullableString {
		t.Errorf("a char column that looks like a number became %s", frame.Fields[1].Type())
	}
}

func TestToTimeSeriesFrameMakesOneSeriesPerLabel(t *testing.T) {
	at := func(minute int) *time.Time {
		t := time.Date(2026, 10, 12, 10, minute, 0, 0, time.UTC)
		return &t
	}
	str := func(s string) *string { return &s }
	num := func(f float64) *float64 { return &f }
	// Rows out of time order, and one without a time
	long := data.NewFrame("response",
		data.NewField("time", nil, []*time.Time{at(1), at(0), nil, at(0), at(1)}),
		data.NewField("service", nil, []*string{str("cart"), str("cart"), str("cart"), str("pay"), str("pay")}),
		data.NewField("rate", nil, []*float64{num(2), num(1), num(9), num(3), num(4)}),
	)

	wide, err := toTimeSeriesFrame(long)
	if err != nil {
		t.Fatal(err)
	}
	if wide.TimeSeriesSchema().Type != data.TimeSeriesTypeWide {
		t.Fatalf("result is %s, want a wide time series", wide.TimeSeriesSchema().Type)
	}
	if n, _ := wide.RowLen(); n != 2 {
		t.Fatalf("rows = %d, want 2 (one per time)", n)
	}
	values := map[string][]float64{}
	for _, f := range wide.Fields[1:] {
		for i := 0; i < f.Len(); i++ {
			v, _ := f.ConcreteAt(i)
			values[f.Labels["service"]] = append(values[f.Labels["service"]], v.(float64))
		}
	}
	if got := values["cart"]; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("cart = %v, want [1 2]", got)
	}
	if got := values["pay"]; len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Errorf("pay = %v, want [3 4]", got)
	}
}

func TestToTimeSeriesFrameLeavesOtherFramesAlone(t *testing.T) {
	table := data.NewFrame("response",
		data.NewField("service", nil, []string{"cart"}),
		data.NewField("requests", nil, []int64{3}),
	)
	got, err := toTimeSeriesFrame(table)
	if err != nil || got != table {
		t.Errorf("toTimeSeriesFrame changed a frame that is not a long time series: %v", err)
	}
}
