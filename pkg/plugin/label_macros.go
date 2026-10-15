package plugin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Label macros filter rows on a label held in a JSON column with the
// text_search property, such as the labels of the metric_series table the
// Kinetica OpenTelemetry exporter writes:
//
//	$__labelMatch(column, 'key', values)    the label equals one of the values
//	$__labelRegex(column, 'key', patterns)  the label matches one of the regular expressions
//
// values and patterns are a JSON string or array of strings, which is how
// ${var:json} writes a dashboard variable's value or values, so a multi-value
// variable can be passed whole. Empty values are ignored; no values left, an
// empty array, or a bare * (a variable's custom "All" value) means no filter.
//
// A macro expands to one text_match call, or to 1 = 1 for no filter:
//
//	$__labelMatch(s.labels, 'http.route', ["/a","/b"])
//	  -> text_match(s.labels, 'labels.http.route:"/a" OR labels.http.route:"/b"')
//	$__labelRegex(labels, 'service.name', "check.*")
//	  -> text_match(labels, 'labels.service.name:/check.*/')
//
// A value is matched exactly and case-sensitively when the column's text index
// uses the keyword analyzer. A pattern is matched against the whole value.

const (
	labelMatchMacro = "$__labelMatch("
	labelRegexMacro = "$__labelRegex("
	noLabelFilter   = "1 = 1"
)

var (
	// A column, optionally qualified (alias.column), each part optionally quoted
	labelColumnPattern = regexp.MustCompile(`^"?[A-Za-z_][A-Za-z0-9_]*"?(\."?[A-Za-z_][A-Za-z0-9_]*"?)*$`)
	// The keys a term query can address; a regex also takes no hyphen
	labelMatchKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)
	labelRegexKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
)

// expandLabelMacros - the query with its label macros expanded
func expandLabelMacros(sql string) (string, error) {
	var out strings.Builder
	for {
		start, macro := nextLabelMacro(sql)
		if start < 0 {
			out.WriteString(sql)
			return out.String(), nil
		}
		argsStart := start + len(macro)
		args, end, err := macroArguments(sql, argsStart)
		if err != nil {
			return "", fmt.Errorf("%s: %w", strings.TrimSuffix(macro, "("), err)
		}
		expansion, err := expandLabelMacro(macro, args)
		if err != nil {
			return "", fmt.Errorf("%s: %w", strings.TrimSuffix(macro, "("), err)
		}
		out.WriteString(sql[:start])
		out.WriteString(expansion)
		sql = sql[end:]
	}
}

// nextLabelMacro - the position of the first label macro in sql, -1 if none,
// and which macro it is
func nextLabelMacro(sql string) (int, string) {
	match, regex := strings.Index(sql, labelMatchMacro), strings.Index(sql, labelRegexMacro)
	switch {
	case match < 0 && regex < 0:
		return -1, ""
	case regex < 0 || (match >= 0 && match < regex):
		return match, labelMatchMacro
	default:
		return regex, labelRegexMacro
	}
}

// macroArguments - the comma-separated arguments of a macro call whose
// argument list starts at sql[from], up to its closing parenthesis. Commas and
// parentheses inside SQL strings ('...'), JSON strings ("...") and brackets do
// not end an argument.
//
//	@return []string - the arguments, trimmed
//	@return int - the position just after the closing parenthesis
func macroArguments(sql string, from int) ([]string, int, error) {
	var args []string
	depth := 0 // parentheses and brackets opened within the arguments
	argStart := from
	for i := from; i < len(sql); i++ {
		switch c := sql[i]; c {
		case '\'':
			// A SQL string; '' is a quote within it
			for i++; i < len(sql); i++ {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i++
						continue
					}
					break
				}
			}
		case '"':
			// A JSON string, or a quoted identifier; \ escapes within it
			for i++; i < len(sql) && sql[i] != '"'; i++ {
				if sql[i] == '\\' {
					i++
				}
			}
		case '(', '[':
			depth++
		case ']':
			depth--
		case ')':
			if depth == 0 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				return args, i + 1, nil
			}
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				argStart = i + 1
			}
		}
	}
	return nil, 0, fmt.Errorf("no closing parenthesis")
}

// expandLabelMacro - the SQL of one label macro call
func expandLabelMacro(macro string, args []string) (string, error) {
	if len(args) != 3 {
		return "", fmt.Errorf("takes 3 arguments, a column, a quoted label key and a JSON string or array, not %d", len(args))
	}
	column := args[0]
	if !labelColumnPattern.MatchString(column) {
		return "", fmt.Errorf("%q is not a column name", column)
	}
	key, err := sqlStringLiteral(args[1])
	if err != nil {
		return "", fmt.Errorf("the label key: %w", err)
	}
	values, err := labelValues(args[2])
	if err != nil {
		return "", err
	}
	if len(values) == 0 {
		return noLabelFilter, nil
	}

	// The text search addresses a JSON column's keys through the column's own
	// name, without a table alias or quotes
	parts := strings.Split(column, ".")
	path := strings.Trim(parts[len(parts)-1], `"`) + "." + key

	var expression string
	if macro == labelMatchMacro {
		if !labelMatchKeyPattern.MatchString(key) {
			return "", fmt.Errorf("%q is not a label key: use letters, digits, _, . and -", key)
		}
		terms := make([]string, len(values))
		for i, v := range values {
			terms[i] = path + `:"` + escapeSearchPhrase(v) + `"`
		}
		expression = strings.Join(terms, " OR ")
	} else {
		if !labelRegexKeyPattern.MatchString(key) {
			return "", fmt.Errorf("%q is not a label key a regex can match: use letters, digits, _ and .", key)
		}
		pattern := values[0]
		if len(values) > 1 {
			pattern = "(" + strings.Join(values, "|") + ")"
		}
		// A regular expression on a JSON key must be the whole search expression
		expression = path + ":/" + pattern + "/"
	}
	return "text_match(" + column + ", '" + strings.ReplaceAll(expression, "'", "''") + "')", nil
}

// sqlStringLiteral - the text of a single-quoted SQL string literal
func sqlStringLiteral(arg string) (string, error) {
	if len(arg) < 2 || arg[0] != '\'' || arg[len(arg)-1] != '\'' {
		return "", fmt.Errorf("%s is not a quoted string such as 'service.name'", arg)
	}
	return strings.ReplaceAll(arg[1:len(arg)-1], "''", "'"), nil
}

// labelValues - the values of a JSON string or array of strings (or numbers or
// Booleans, written as text), without empty ones; none for a bare *
func labelValues(arg string) ([]string, error) {
	if arg == "*" {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(arg), &decoded); err != nil {
		return nil, fmt.Errorf("the values must be a JSON string or array, as ${var:json} writes them, not %s", arg)
	}
	var items []any
	switch v := decoded.(type) {
	case []any:
		if len(v) == 0 {
			return nil, nil
		}
		items = v
	default:
		items = []any{v}
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			if v != "" {
				values = append(values, v)
			}
		case float64, bool:
			values = append(values, fmt.Sprint(v))
		default:
			return nil, fmt.Errorf("the values must be strings, not %s", arg)
		}
	}
	return values, nil
}

// escapeSearchPhrase - a value written inside a quoted search phrase, where a
// backslash and a double quote must be escaped
func escapeSearchPhrase(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}
