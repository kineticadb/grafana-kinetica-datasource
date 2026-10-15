package plugin

import (
	"strings"
	"testing"
)

func TestExpandLabelMacros(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"one value",
			`SELECT 1 FROM s WHERE $__labelMatch(labels, 'service.name', "checkout")`,
			`SELECT 1 FROM s WHERE text_match(labels, 'labels.service.name:"checkout"')`,
		},
		{
			"several values, as ${var:json} writes a multi-value variable",
			`WHERE $__labelMatch(labels, 'http.route', ["/api/v1/cart","/api/v1/pay"])`,
			`WHERE text_match(labels, 'labels.http.route:"/api/v1/cart" OR labels.http.route:"/api/v1/pay"')`,
		},
		{
			"a qualified column keeps its alias in SQL but not in the search path",
			`WHERE $__labelMatch(s.labels, 'host.name', ["h1"])`,
			`WHERE text_match(s.labels, 'labels.host.name:"h1"')`,
		},
		{
			"a quoted column",
			`WHERE $__labelMatch("s"."labels", 'k', "v")`,
			`WHERE text_match("s"."labels", 'labels.k:"v"')`,
		},
		{
			"quotes, backslashes and apostrophes are escaped",
			`WHERE $__labelMatch(labels, 'k', ["say \"hi\"", "back\\slash", "o'brien"])`,
			`WHERE text_match(labels, 'labels.k:"say \"hi\"" OR labels.k:"back\\slash" OR labels.k:"o''brien"')`,
		},
		{
			"commas and parentheses inside values do not end the arguments",
			`WHERE $__labelMatch(labels, 'k', ["a, b)", "(c]"]) AND x = 1`,
			`WHERE text_match(labels, 'labels.k:"a, b)" OR labels.k:"(c]"') AND x = 1`,
		},
		{
			"a key with a hyphen",
			`WHERE $__labelMatch(labels, 'k8s-pod', "p1")`,
			`WHERE text_match(labels, 'labels.k8s-pod:"p1"')`,
		},
		{
			"numbers are matched as text",
			`WHERE $__labelMatch(labels, 'http.response.status_code', [500, 503])`,
			`WHERE text_match(labels, 'labels.http.response.status_code:"500" OR labels.http.response.status_code:"503"')`,
		},
		{"all values: a bare *", `WHERE $__labelMatch(labels, 'k', *)`, `WHERE 1 = 1`},
		{"no values: an empty array", `WHERE $__labelMatch(labels, 'k', [])`, `WHERE 1 = 1`},
		{"only empty values", `WHERE $__labelMatch(labels, 'k', ["", ""])`, `WHERE 1 = 1`},
		{
			"one regex",
			`WHERE $__labelRegex(labels, 'service.name', "check.*")`,
			`WHERE text_match(labels, 'labels.service.name:/check.*/')`,
		},
		{
			"several regexes become one alternation",
			`WHERE $__labelRegex(labels, 'http.route', ["/api/.*", "/health"])`,
			`WHERE text_match(labels, 'labels.http.route:/(/api/.*|/health)/')`,
		},
		{"an empty regex is no filter", `WHERE $__labelRegex(labels, 'k', "")`, `WHERE 1 = 1`},
		{
			"two macros in one query",
			`WHERE $__labelRegex(labels, 'a', "x.*") AND $__labelMatch(labels, 'b', "y")`,
			`WHERE text_match(labels, 'labels.a:/x.*/') AND text_match(labels, 'labels.b:"y"')`,
		},
		{"no macro", `SELECT 1`, `SELECT 1`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := expandLabelMacros(c.in)
			if err != nil {
				t.Fatalf("expandLabelMacros(%s): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("expandLabelMacros:\n  in %s\n got %s\nwant %s", c.in, got, c.want)
			}
		})
	}
}

func TestExpandLabelMacrosRejectsBadArguments(t *testing.T) {
	cases := map[string]string{
		`$__labelMatch(labels, 'k')`:               "takes 3 arguments",
		`$__labelMatch(labels, 'k', "v", "w")`:     "takes 3 arguments",
		`$__labelMatch(labels, k, "v")`:            "not a quoted string",
		`$__labelMatch(labels, 'k', v)`:            "JSON string or array",
		`$__labelMatch(labels, 'k', {"a": 1})`:     "must be strings",
		`$__labelMatch(labels, 'k8s-pod''s', "v")`: "not a label key",
		`$__labelMatch(labels, 'k', [{"a": 1}])`:   "must be strings",
		`$__labelMatch(1labels, 'k', "v")`:         "not a column name",
		`$__labelMatch(labels, 'k k', "v")`:        "not a label key",
		`$__labelRegex(labels, 'k8s-pod', "p.*")`:  "not a label key a regex can match",
		`$__labelMatch(labels, 'k', ["unclosed")`:  "no closing parenthesis",
		`SELECT $__labelMatch(labels, 'k', "v"`:    "no closing parenthesis",
	}
	for in, want := range cases {
		_, err := expandLabelMacros(in)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("expandLabelMacros(%s) = %v, want an error containing %q", in, err, want)
		}
	}
}
