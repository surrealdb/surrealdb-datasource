package plugin

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

// surrealDateTime formats a time as a SurrealDB datetime literal, e.g.
// d'2023-11-27T22:30:23Z'. This is the form SurrealQL expects for datetime
// comparisons, unlike the plain quoted strings the default SQL macros emit.
func surrealDateTime(t time.Time) string {
	return fmt.Sprintf("d'%s'", t.UTC().Format(time.RFC3339))
}

// macros overrides the time-related sqlutil macros so they produce SurrealQL
// rather than ANSI SQL. sqlutil.Interpolate merges these over
// sqlutil.DefaultMacros, so the remaining defaults ($__interval, $__interval_ms,
// $__table, $__column) stay available unchanged.
//
//	$__timeFrom(col)        => col >= d'<from>'
//	$__timeTo(col)          => col <= d'<to>'
//	$__timeFilter(col)      => col >= d'<from>' AND col <= d'<to>'
//	$__timeGroup(col, unit) => time::group(col, 'unit')
var macros = sqlutil.Macros{
	"timeFrom": func(query *sqlutil.Query, args []string) (string, error) {
		col, err := singleColumnArg("timeFrom", args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s >= %s", col, surrealDateTime(query.TimeRange.From)), nil
	},
	"timeTo": func(query *sqlutil.Query, args []string) (string, error) {
		col, err := singleColumnArg("timeTo", args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s <= %s", col, surrealDateTime(query.TimeRange.To)), nil
	},
	"timeFilter": func(query *sqlutil.Query, args []string) (string, error) {
		col, err := singleColumnArg("timeFilter", args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			"%s >= %s AND %s <= %s",
			col, surrealDateTime(query.TimeRange.From),
			col, surrealDateTime(query.TimeRange.To),
		), nil
	},
	"timeGroup": func(_ *sqlutil.Query, args []string) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("$__timeGroup: expected 2 arguments, got %d", len(args))
		}
		return fmt.Sprintf("time::group(%s, '%s')", strings.TrimSpace(args[0]), strings.TrimSpace(args[1])), nil
	},
}

func singleColumnArg(name string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("$__%s: expected 1 argument, got %d", name, len(args))
	}
	return strings.TrimSpace(args[0]), nil
}
