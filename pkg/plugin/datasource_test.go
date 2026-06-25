package plugin_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grafana-labs/surrealdb-datasource/internal/mocks"
	"github.com/grafana-labs/surrealdb-datasource/pkg/client"
	"github.com/grafana-labs/surrealdb-datasource/pkg/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// config is the shared datasource configuration used across the plugin tests.
var config = client.SurrealConfig{
	Database:  "grafana_ds_tests",
	Endpoint:  "ws://localhost:8000/rpc",
	Namespace: "grafana",
	Username:  "grafana",
}

func TestNewDatasource_InvalidJSON(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData:                json.RawMessage(`invalid json`),
		DecryptedSecureJSONData: map[string]string{"password": "password"},
	}

	if _, err := plugin.NewDatasource(context.Background(), settings); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestQueryData(t *testing.T) {
	ds := plugin.NewDatasourceInstance(client.Use(&mocks.MockSurrealDBClient{
		QueryFunc: func(_ context.Context, _ string, _ map[string]any) ([]client.QueryResult, error) {
			return []client.QueryResult{{Status: "OK", Rows: []map[string]any{{"a": "1"}}}}, nil
		},
	}), &config)

	req := backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "query1", JSON: []byte(`{"rawSql":"SELECT 1"}`)},
			{RefID: "query2", JSON: []byte(`{"rawSql":"SELECT 2"}`)},
		},
	}

	response, err := ds.QueryData(context.Background(), &req)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if response == nil || len(response.Responses) != 2 {
		t.Errorf("expected 2 responses, got %v", response)
	}
}

func TestCheckHealth_Success(t *testing.T) {
	var gotSQL string
	ds := plugin.NewDatasourceInstance(client.Use(&mocks.MockSurrealDBClient{
		QueryFunc: func(_ context.Context, sql string, _ map[string]any) ([]client.QueryResult, error) {
			gotSQL = sql
			return []client.QueryResult{{Status: "OK"}}, nil
		},
	}), &config)

	result, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result == nil || result.Status != backend.HealthStatusOk {
		t.Errorf("expected health ok, got %+v", result)
	}
	if gotSQL != "RETURN 1;" {
		t.Errorf("expected probe query 'RETURN 1;', got %q", gotSQL)
	}
}

func TestCheckHealth_Error(t *testing.T) {
	ds := plugin.NewDatasourceInstance(client.Use(&mocks.MockSurrealDBClient{
		QueryFunc: func(_ context.Context, _ string, _ map[string]any) ([]client.QueryResult, error) {
			return nil, errors.New("query error")
		},
	}), &config)

	result, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result == nil || result.Status != backend.HealthStatusError {
		t.Errorf("expected health error, got %+v", result)
	}
}
