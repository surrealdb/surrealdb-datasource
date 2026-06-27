package client

import (
	"reflect"
	"testing"
)

// TestBuildAuth checks that the sign-in payload includes the namespace and
// database only as appropriate for the authentication level, since SurrealDB
// infers the level from the fields present (a root user that signs in with a
// namespace/database set is rejected).
func TestBuildAuth(t *testing.T) {
	base := SurrealConfig{Namespace: "ns", Database: "db", Username: "u", Password: "p"}

	cases := []struct {
		name          string
		scope         string
		access        string
		wantNamespace string
		wantDatabase  string
		wantAccess    string
	}{
		{name: "root by default", scope: "", wantNamespace: "", wantDatabase: ""},
		{name: "explicit root", scope: AuthScopeRoot, wantNamespace: "", wantDatabase: ""},
		{name: "namespace user", scope: AuthScopeNamespace, wantNamespace: "ns", wantDatabase: ""},
		{name: "database user", scope: AuthScopeDatabase, wantNamespace: "ns", wantDatabase: "db"},
		{name: "record access", scope: AuthScopeRoot, access: "account", wantNamespace: "ns", wantDatabase: "db", wantAccess: "account"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.AuthScope = tc.scope
			cfg.Access = tc.access

			auth := buildAuth(&cfg)

			if auth.Username != "u" || auth.Password != "p" {
				t.Errorf("expected credentials passed through, got user=%q", auth.Username)
			}
			if auth.Namespace != tc.wantNamespace {
				t.Errorf("namespace: got %q, want %q", auth.Namespace, tc.wantNamespace)
			}
			if auth.Database != tc.wantDatabase {
				t.Errorf("database: got %q, want %q", auth.Database, tc.wantDatabase)
			}
			if auth.Access != tc.wantAccess {
				t.Errorf("access: got %q, want %q", auth.Access, tc.wantAccess)
			}
		})
	}
}

// TestNormalizeRows covers the result shapes SurrealDB can return for a single
// statement. Only an array of objects maps directly to rows; objects, scalars
// (e.g. RETURN 1 from the health check) and scalar arrays must be wrapped so
// they don't fail to render.
func TestNormalizeRows(t *testing.T) {
	cases := []struct {
		name   string
		result any
		want   []map[string]any
	}{
		{"nil/NONE", nil, nil},
		{
			"array of objects",
			[]any{map[string]any{"a": 1}, map[string]any{"a": 2}},
			[]map[string]any{{"a": 1}, {"a": 2}},
		},
		{
			"single object",
			map[string]any{"a": 1},
			[]map[string]any{{"a": 1}},
		},
		{
			"scalar (RETURN 1)",
			uint64(1),
			[]map[string]any{{"value": uint64(1)}},
		},
		{
			"array of scalars (SELECT VALUE)",
			[]any{"alice", "bob"},
			[]map[string]any{{"value": "alice"}, {"value": "bob"}},
		},
		{
			"empty array",
			[]any{},
			[]map[string]any{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRows(tc.result); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("normalizeRows(%#v) = %#v, want %#v", tc.result, got, tc.want)
			}
		})
	}
}
