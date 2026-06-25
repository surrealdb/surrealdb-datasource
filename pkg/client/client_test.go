package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/surrealdb/surrealdb-datasource/internal/mocks"
	"github.com/surrealdb/surrealdb-datasource/pkg/client"
)

func testConfig() *client.SurrealConfig {
	return &client.SurrealConfig{
		Database:  "test_db",
		Endpoint:  "ws://localhost:8000",
		Namespace: "test-namespace",
		Password:  "password",
		Username:  "username",
	}
}

func TestConnect_Success(t *testing.T) {
	var gotConfig *client.SurrealConfig
	var gotNamespace, gotDatabase string

	mockDB := mocks.MockSurrealDBClient{
		SignInFunc: func(_ context.Context, config *client.SurrealConfig) (string, error) {
			gotConfig = config
			return "token", nil
		},
		UseFunc: func(_ context.Context, namespace, database string) error {
			gotNamespace, gotDatabase = namespace, database
			return nil
		},
	}

	c := client.Use(&mockDB)

	if err := c.Connect(context.Background(), testConfig()); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if gotConfig == nil || gotConfig.Username != "username" || gotConfig.Password != "password" {
		t.Errorf("expected sign-in to receive the configured credentials, got %+v", gotConfig)
	}
	if gotNamespace != "test-namespace" || gotDatabase != "test_db" {
		t.Errorf("expected Use(test-namespace, test_db), got Use(%q, %q)", gotNamespace, gotDatabase)
	}
}

func TestConnect_SignInError(t *testing.T) {
	mockDB := mocks.MockSurrealDBClient{
		SignInFunc: func(_ context.Context, _ *client.SurrealConfig) (string, error) {
			return "", errors.New("signin error")
		},
		UseFunc: func(_ context.Context, _, _ string) error {
			t.Error("Use should not be called when sign-in fails")
			return nil
		},
	}

	c := client.Use(&mockDB)

	if err := c.Connect(context.Background(), testConfig()); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestConnect_UseError(t *testing.T) {
	mockDB := mocks.MockSurrealDBClient{
		SignInFunc: func(_ context.Context, _ *client.SurrealConfig) (string, error) {
			return "token", nil
		},
		UseFunc: func(_ context.Context, _, _ string) error {
			return errors.New("use error")
		},
	}

	c := client.Use(&mockDB)

	if err := c.Connect(context.Background(), testConfig()); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestClient_Query(t *testing.T) {
	want := []client.QueryResult{{Status: "OK", Rows: []map[string]any{{"a": int64(1)}}}}

	var gotSQL string
	var gotVars map[string]any
	mockDB := mocks.MockSurrealDBClient{
		QueryFunc: func(_ context.Context, sql string, vars map[string]any) ([]client.QueryResult, error) {
			gotSQL, gotVars = sql, vars
			return want, nil
		},
	}

	c := client.Use(&mockDB)

	got, err := c.Query(context.Background(), "SELECT 1", map[string]any{"x": 1})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotSQL != "SELECT 1" {
		t.Errorf("expected sql to be passed through, got %q", gotSQL)
	}
	if gotVars["x"] != 1 {
		t.Errorf("expected vars to be passed through, got %v", gotVars)
	}
	if len(got) != 1 || got[0].Status != "OK" {
		t.Errorf("expected the query result to be returned, got %+v", got)
	}
}

func TestClient_Close(t *testing.T) {
	closed := false
	mockDB := mocks.MockSurrealDBClient{
		CloseFunc: func(_ context.Context) error {
			closed = true
			return nil
		},
	}

	c := client.Use(&mockDB)

	if err := c.Close(context.Background()); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !closed {
		t.Error("expected Close to be delegated to the underlying client")
	}
}
