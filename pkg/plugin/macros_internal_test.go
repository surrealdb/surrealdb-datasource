package plugin

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

func interpolate(t *testing.T, raw string) string {
	t.Helper()
	query := &sqlutil.Query{
		RawSQL: raw,
		TimeRange: backend.TimeRange{
			From: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			To:   time.Date(2024, 1, 2, 4, 4, 5, 0, time.UTC),
		},
	}
	out, err := sqlutil.Interpolate(query, macros)
	if err != nil {
		t.Fatalf("interpolate %q: %v", raw, err)
	}
	return out
}

func TestSurrealTimeMacros(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "timeFrom emits a SurrealDB datetime literal",
			raw:  "$__timeFrom(ts)",
			want: "ts >= d'2024-01-02T03:04:05Z'",
		},
		{
			name: "timeTo emits a SurrealDB datetime literal",
			raw:  "$__timeTo(ts)",
			want: "ts <= d'2024-01-02T04:04:05Z'",
		},
		{
			name: "timeFilter brackets the range",
			raw:  "$__timeFilter(created_at)",
			want: "created_at >= d'2024-01-02T03:04:05Z' AND created_at <= d'2024-01-02T04:04:05Z'",
		},
		{
			name: "timeGroup uses time::group",
			raw:  "$__timeGroup(ts, day)",
			want: "time::group(ts, 'day')",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := interpolate(t, tc.raw); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
