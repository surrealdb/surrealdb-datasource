package client

import (
	"reflect"
	"testing"
)

// TestBuildAuth_OmitsNamespaceAndDatabase documents the authentication model:
// the namespace and database are selected via Use, not sent in the sign-in
// payload, because SurrealDB infers the auth level from the fields present and
// rejects a root user that signs in with a namespace/database set.
func TestBuildAuth_OmitsNamespaceAndDatabase(t *testing.T) {
	auth := buildAuth(&SurrealConfig{
		Namespace: "ns",
		Database:  "db",
		Username:  "root",
		Password:  "secret",
		Access:    "account",
	})

	if auth.Namespace != "" {
		t.Errorf("expected namespace to be omitted from sign-in, got %q", auth.Namespace)
	}
	if auth.Database != "" {
		t.Errorf("expected database to be omitted from sign-in, got %q", auth.Database)
	}
	if auth.Username != "root" || auth.Password != "secret" {
		t.Errorf("expected credentials to be passed through, got user=%q", auth.Username)
	}
	if auth.Access != "account" {
		t.Errorf("expected access method to be passed through, got %q", auth.Access)
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
