// Package oteldashboards tests the Grafana dashboards of the Kinetica
// OpenTelemetry exporter's metric tables, which are kept in this repository
// but are not part of the plugin's package; see README.md.
package oteldashboards

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// The plugin, and the release of it whose features the dashboards use
const (
	pluginID      = "kinetica-grafana-datasource"
	pluginVersion = "1.1.0"
)

var dashboardGlob = filepath.Join("dashboards", "*.json")

var (
	// A variable or macro reference: ${name}, ${name:format} or $name
	variableReferencePattern = regexp.MustCompile(`\$\{(\w+)(?::(\w+))?\}|\$(\w+)`)
	// The macros of the plugin and the Grafana variables the dashboards use
	pluginMacroNames = []string{"__timeFilter", "__timeGroup", "__labelMatch", "__labelRegex", "__interval"}
	// A table read with a schema: FROM schema.table or JOIN schema.table
	qualifiedTablePattern = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([A-Za-z_$][\w$]*)\.`)
)

// dashboard - the parts of a dashboard the tests read
type dashboard struct {
	UID      string `json:"uid"`
	Title    string `json:"title"`
	Requires []struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"__requires"`
	Templating struct {
		List []variable `json:"list"`
	} `json:"templating"`
	Panels []panel `json:"panels"`
}

type variable struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Query      json.RawMessage `json:"query"`
	Multi      bool            `json:"multi"`
	IncludeAll bool            `json:"includeAll"`
	AllValue   string          `json:"allValue"`
	Datasource *struct {
		Type string `json:"type"`
		UID  string `json:"uid"`
	} `json:"datasource"`
}

// queryText - a datasource or textbox variable's query, as text
func (v variable) queryText() string {
	var text string
	_ = json.Unmarshal(v.Query, &text)
	return text
}

// rawSQL - a query variable's SQL
func (v variable) rawSQL() string {
	var query struct {
		RawSQL string `json:"rawSql"`
	}
	_ = json.Unmarshal(v.Query, &query)
	return query.RawSQL
}

type panel struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Interval   string `json:"interval"`
	Datasource *struct {
		Type string `json:"type"`
		UID  string `json:"uid"`
	} `json:"datasource"`
	Targets []struct {
		RefID  string `json:"refId"`
		Format string `json:"format"`
		RawSQL string `json:"rawSql"`
	} `json:"targets"`
	Transformations []json.RawMessage `json:"transformations"`
}

// loadDashboards - the dashboards, by file name
func loadDashboards(t *testing.T) map[string]dashboard {
	t.Helper()
	paths, err := filepath.Glob(dashboardGlob)
	if err != nil || len(paths) == 0 {
		t.Fatalf("no dashboards in %s: %v", dashboardGlob, err)
	}
	dashboards := map[string]dashboard{}
	for _, path := range paths {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var d dashboard
		if err := json.Unmarshal(text, &d); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		dashboards[filepath.Base(path)] = d
	}
	return dashboards
}

// dashboardQuery - a query of a dashboard: a panel's, or a query variable's
type dashboardQuery struct {
	file   string
	title  string // the panel's title, or "variable <name>"
	panel  *panel
	format string // the target's format; empty for a variable, which is a table
	sql    string
}

// name - the query's file and title
func (q dashboardQuery) name() string {
	return q.file + "/" + q.title
}

// dashboardQueries - every query of the dashboards, in a stable order
func dashboardQueries(t *testing.T) []dashboardQuery {
	t.Helper()
	dashboards := loadDashboards(t)
	files := make([]string, 0, len(dashboards))
	for file := range dashboards {
		files = append(files, file)
	}
	slices.Sort(files)

	var queries []dashboardQuery
	for _, file := range files {
		d := dashboards[file]
		for _, v := range d.Templating.List {
			if v.Type == "query" {
				queries = append(queries, dashboardQuery{file: file, title: "variable " + v.Name, sql: v.rawSQL()})
			}
		}
		for i := range d.Panels {
			p := &d.Panels[i]
			for _, target := range p.Targets {
				queries = append(queries, dashboardQuery{file: file, title: p.Title, panel: p, format: target.Format,
					sql: target.RawSQL})
			}
		}
	}
	return queries
}

// grafanaVariables - the values Grafana interpolates into a dashboard query
// before it reaches the plugin
type grafanaVariables struct {
	schema string
	// values are the values of the multi-value variables; a variable without
	// one is at All, whose custom value is *
	values   map[string][]string
	interval string // what Grafana gives $__interval; 1m when empty
}

// interpolate - a dashboard query as Grafana hands it to the plugin: $schema
// as it is, ${name:json} as a JSON array of the variable's values (Grafana
// inserts a custom All value as it is), $__interval as a duration, and the
// plugin's macros left for the plugin
func (v grafanaVariables) interpolate(t *testing.T, sql string) string {
	t.Helper()
	interval := v.interval
	if interval == "" {
		interval = "1m"
	}
	return variableReferencePattern.ReplaceAllStringFunc(sql, func(ref string) string {
		m := variableReferencePattern.FindStringSubmatch(ref)
		name, format := m[1]+m[3], m[2]
		switch {
		case name == "schema":
			return v.schema
		case name == "__interval":
			return interval
		case slices.Contains(pluginMacroNames, name):
			return ref
		case format == "json":
			values, ok := v.values[name]
			if !ok {
				return "*"
			}
			// As JSON.stringify writes it
			var text bytes.Buffer
			encoder := json.NewEncoder(&text)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(values); err != nil {
				t.Fatal(err)
			}
			return string(bytes.TrimSpace(text.Bytes()))
		}
		t.Fatalf("no value for %s", ref)
		return ""
	})
}

func TestDashboards(t *testing.T) {
	uids := map[string]string{}
	for file, d := range loadDashboards(t) {
		t.Run(file, func(t *testing.T) {
			if d.UID == "" {
				t.Fatal("no uid")
			}
			if other, ok := uids[d.UID]; ok {
				t.Errorf("uid %s is also the uid of %s", d.UID, other)
			}
			uids[d.UID] = file

			var requiresPlugin bool
			for _, r := range d.Requires {
				if r.Type == "datasource" && r.ID == pluginID {
					requiresPlugin = true
					if r.Version != pluginVersion {
						t.Errorf("__requires has the plugin %s, want %s", r.Version, pluginVersion)
					}
				}
			}
			if !requiresPlugin {
				t.Error("__requires does not name the plugin")
			}

			// The datasource and schema variables, then query variables
			variables := d.Templating.List
			if len(variables) < 2 {
				t.Fatalf("%d variables, want the datasource and the schema first", len(variables))
			}
			if ds := variables[0]; ds.Name != "DS_KINETICA" || ds.Type != "datasource" || ds.queryText() != pluginID {
				t.Errorf("first variable is %s (%s, %s), want DS_KINETICA, a datasource of %s",
					ds.Name, ds.Type, ds.queryText(), pluginID)
			}
			if schema := variables[1]; schema.Name != "schema" || schema.Type != "textbox" || schema.queryText() != "otel" {
				t.Errorf("second variable is %s (%s, %s), want schema, a textbox of otel",
					schema.Name, schema.Type, schema.queryText())
			}

			defined := []string{"schema"}
			for _, v := range variables[2:] {
				t.Run("variable "+v.Name, func(t *testing.T) {
					if v.Type != "query" {
						t.Errorf("type %s, want query", v.Type)
					}
					if v.Datasource == nil || v.Datasource.UID != "${DS_KINETICA}" {
						t.Errorf("datasource %v, want ${DS_KINETICA}", v.Datasource)
					}
					if !v.Multi || !v.IncludeAll {
						t.Error("not a multi-value variable with All")
					}
					if v.AllValue != "*" {
						t.Errorf("All is %q, want *, which $__labelMatch takes as no filter", v.AllValue)
					}
					if v.rawSQL() == "" {
						t.Fatal("no query")
					}
					// A variable's query may use the variables before it
					checkVariableReferences(t, v.rawSQL(), defined)
				})
				defined = append(defined, v.Name)
			}

			for _, p := range d.Panels {
				if p.Type == "text" {
					continue
				}
				t.Run(p.Title, func(t *testing.T) {
					if p.Datasource == nil || p.Datasource.Type != pluginID || p.Datasource.UID != "${DS_KINETICA}" {
						t.Errorf("datasource %v, want ${DS_KINETICA} of %s", p.Datasource, pluginID)
					}
					if len(p.Targets) != 1 {
						t.Fatalf("%d targets, want 1", len(p.Targets))
					}
					target := p.Targets[0]
					checkVariableReferences(t, target.RawSQL, defined)

					switch p.Type {
					case "timeseries":
						// The plugin makes one series per text column value,
						// and recognises the time bucket by its type
						if target.Format != "time_series" {
							t.Errorf("format %q, want time_series", target.Format)
						}
						if len(p.Transformations) != 0 {
							t.Error("transformations, which the time_series format makes unneeded")
						}
						if p.Interval != "1m" {
							t.Errorf("min interval %q, want buckets of a minute at least", p.Interval)
						}
						if !regexp.MustCompile(`\$__timeGroup\(`).MatchString(target.RawSQL) ||
							!regexp.MustCompile(`\bAS time\b`).MatchString(target.RawSQL) {
							t.Error("no $__timeGroup(...) AS time")
						}
					case "table":
						if target.Format != "table" {
							t.Errorf("format %q, want table", target.Format)
						}
					default:
						t.Errorf("unexpected panel type %s", p.Type)
					}
				})
			}
		})
	}
}

// checkVariableReferences - checks that a query reads its tables from
// $schema and uses only the given variables, the multi-value ones as
// ${name:json}, and the plugin's macros
func checkVariableReferences(t *testing.T, sql string, defined []string) {
	t.Helper()
	if regexp.MustCompile(`\botel\.metric_`).MatchString(sql) {
		t.Error("a table read from otel rather than $schema")
	}
	for _, m := range qualifiedTablePattern.FindAllStringSubmatch(sql, -1) {
		if m[1] != "$schema" {
			t.Errorf("a table read from %s rather than $schema", m[1])
		}
	}
	for _, m := range variableReferencePattern.FindAllStringSubmatch(sql, -1) {
		name, format := m[1]+m[3], m[2]
		switch {
		case slices.Contains(pluginMacroNames, name):
		case name == "schema":
			if format != "" {
				t.Errorf("%s: $schema is used as it is", m[0])
			}
		case slices.Contains(defined, name):
			if format != "json" {
				t.Errorf("%s: a multi-value variable is passed as ${%s:json}", m[0], name)
			}
		default:
			t.Errorf("%s is not a variable of the dashboard", m[0])
		}
	}
}

func TestGrafanaVariables(t *testing.T) {
	vars := grafanaVariables{schema: "otel", values: map[string][]string{"service": {"checkout", `say "hi"`, "a&b"}}}
	for in, want := range map[string]string{
		"SELECT * FROM $schema.metric_series":                "SELECT * FROM otel.metric_series",
		"$__timeGroup(time_unix, $__interval) AS time":       "$__timeGroup(time_unix, 1m) AS time",
		"WHERE $__timeFilter(p.time_unix)":                   "WHERE $__timeFilter(p.time_unix)",
		`JSON_VALUE(labels, '$["host.name"]')`:               `JSON_VALUE(labels, '$["host.name"]')`,
		"$__labelMatch(labels, 'http.route', ${route:json})": "$__labelMatch(labels, 'http.route', *)",
		"$__labelMatch(labels, 'service.name', ${service:json})": `$__labelMatch(labels, 'service.name', ` +
			`["checkout","say \"hi\"","a&b"])`,
	} {
		if got := vars.interpolate(t, in); got != want {
			t.Errorf("%s\n got %s\nwant %s", in, got, want)
		}
	}
}

// TestGenerator checks that the dashboards are what generate_dashboards.py
// writes, so that a change made to their JSON alone is not lost
func TestGenerator(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	out := t.TempDir()
	if text, err := exec.Command(python, "generate_dashboards.py", out).CombinedOutput(); err != nil {
		t.Fatalf("generate_dashboards.py: %v\n%s", err, text)
	}
	dashboards := loadDashboards(t)
	for file := range dashboards {
		generated, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Errorf("%s is not generated: %v", file, err)
			continue
		}
		committed, _ := os.ReadFile(filepath.Join("dashboards", file))
		if !bytes.Equal(generated, committed) {
			t.Errorf("%s differs from what generate_dashboards.py writes; edit the generator and run it", file)
		}
	}
	generated, _ := filepath.Glob(filepath.Join(out, "*.json"))
	for _, path := range generated {
		if _, ok := dashboards[filepath.Base(path)]; !ok {
			t.Errorf("%s is generated but not in dashboards", filepath.Base(path))
		}
	}
}
