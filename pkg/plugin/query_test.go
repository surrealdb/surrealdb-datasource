package plugin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/surrealdb/surrealdb-datasource/internal/mocks"
	"github.com/surrealdb/surrealdb-datasource/pkg/client"
	"github.com/surrealdb/surrealdb-datasource/pkg/plugin"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func dataQuery(rawSQL string) backend.DataQuery {
	return backend.DataQuery{RefID: "A", JSON: []byte(`{"rawSql":"` + rawSQL + `"}`)}
}

func queryMock(results []client.QueryResult, err error) *mocks.MockSurrealDBClient {
	return &mocks.MockSurrealDBClient{
		QueryFunc: func(_ context.Context, _ string, _ map[string]any) ([]client.QueryResult, error) {
			return results, err
		},
	}
}

func respond(t *testing.T, results []client.QueryResult, err error) backend.DataResponse {
	t.Helper()
	ds := plugin.NewDatasourceInstance(client.Use(queryMock(results, err)), &config)
	return ds.CreateDataResponse(context.Background(), dataQuery("SELECT * FROM test"))
}

func fieldsByName(frame *data.Frame) map[string]*data.Field {
	out := map[string]*data.Field{}
	for _, f := range frame.Fields {
		out[f.Name] = f
	}
	return out
}

func TestCreateDataResponse_Success(t *testing.T) {
	results := []client.QueryResult{{
		Status: "OK",
		Rows:   []map[string]any{{"column1": "value1", "column2": "value2"}},
	}}

	response := respond(t, results, nil)

	if response.Error != nil {
		t.Fatalf("unexpected error: %s", response.Error)
	}
	if len(response.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(response.Frames))
	}
	if response.Frames[0].Fields[0].Name != "column1" || response.Frames[0].Fields[1].Name != "column2" {
		t.Errorf("expected fields sorted as column1, column2; got %q, %q",
			response.Frames[0].Fields[0].Name, response.Frames[0].Fields[1].Name)
	}
}

func TestCreateDataResponse_TypedColumns(t *testing.T) {
	dt := models.CustomDateTime{Time: time.Date(2023, 11, 27, 22, 30, 23, 0, time.UTC)}
	results := []client.QueryResult{{
		Status: "OK",
		Rows: []map[string]any{{
			"dt":     dt,
			"int":    uint64(42), // SurrealDB positive integers decode as uint64
			"neg":    int64(-7),
			"float":  3.14,
			"flag":   true,
			"name":   "alice",
			"id":     models.RecordID{Table: "person", ID: "alice"},
			"nested": map[string]any{"city": "London"},
		}},
	}}

	response := respond(t, results, nil)
	if response.Error != nil {
		t.Fatalf("unexpected error: %s", response.Error)
	}
	fields := fieldsByName(response.Frames[0])

	wantTypes := map[string]data.FieldType{
		"dt":     data.FieldTypeNullableTime,
		"int":    data.FieldTypeNullableInt64,
		"neg":    data.FieldTypeNullableInt64,
		"float":  data.FieldTypeNullableFloat64,
		"flag":   data.FieldTypeNullableBool,
		"name":   data.FieldTypeNullableString,
		"id":     data.FieldTypeNullableString,
		"nested": data.FieldTypeNullableString,
	}
	for name, want := range wantTypes {
		if fields[name] == nil {
			t.Errorf("missing field %q", name)
			continue
		}
		if got := fields[name].Type(); got != want {
			t.Errorf("field %q: expected type %v, got %v", name, want, got)
		}
	}

	if got := fields["dt"].At(0).(*time.Time); !got.Equal(dt.Time) {
		t.Errorf("dt: expected %v, got %v", dt.Time, *got)
	}
	if got := fields["id"].At(0).(*string); got == nil || *got != "person:alice" {
		t.Errorf("id: expected 'person:alice', got %v", got)
	}
	if got := fields["nested"].At(0).(*string); got == nil || *got != `{"city":"London"}` {
		t.Errorf("nested: expected JSON string, got %v", got)
	}
}

func TestCreateDataResponse_MixedNumericPromotesToFloat(t *testing.T) {
	results := []client.QueryResult{{
		Status: "OK",
		Rows: []map[string]any{
			{"n": int64(1)},
			{"n": 2.5},
		},
	}}

	response := respond(t, results, nil)
	fields := fieldsByName(response.Frames[0])
	if got := fields["n"].Type(); got != data.FieldTypeNullableFloat64 {
		t.Errorf("expected mixed int/float column to promote to float64, got %v", got)
	}
}

func TestCreateDataResponse_NullAndMissingValues(t *testing.T) {
	results := []client.QueryResult{{
		Status: "OK",
		Rows: []map[string]any{
			{"a": int64(1), "b": "x"},
			{"a": nil}, // b missing, a is NONE/null
		},
	}}

	response := respond(t, results, nil)
	if response.Error != nil {
		t.Fatalf("unexpected error: %s", response.Error)
	}
	fields := fieldsByName(response.Frames[0])
	if fields["a"].Len() != 2 || fields["b"].Len() != 2 {
		t.Fatalf("expected both fields length 2")
	}
	if fields["a"].At(1).(*int64) != nil {
		t.Errorf("expected null for missing/NONE numeric value")
	}
	if fields["b"].At(1).(*string) != nil {
		t.Errorf("expected null for missing string value")
	}
}

func TestCreateDataResponse_EmptyResult(t *testing.T) {
	response := respond(t, []client.QueryResult{{Status: "OK", Rows: nil}}, nil)
	if response.Error != nil {
		t.Fatalf("unexpected error: %s", response.Error)
	}
	if len(response.Frames) != 0 {
		t.Errorf("expected no frames for empty result, got %d", len(response.Frames))
	}
}

func TestCreateDataResponse_MultiStatement(t *testing.T) {
	results := []client.QueryResult{
		{Status: "OK", Rows: []map[string]any{{"a": "1"}}},
		{Status: "OK", Rows: []map[string]any{{"b": "2"}}},
	}

	response := respond(t, results, nil)
	if len(response.Frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(response.Frames))
	}
	if response.Frames[0].Name != "response" || response.Frames[1].Name != "response_1" {
		t.Errorf("expected frame names response, response_1; got %q, %q",
			response.Frames[0].Name, response.Frames[1].Name)
	}
}

func TestCreateDataResponse_StatementError(t *testing.T) {
	results := []client.QueryResult{{Status: "ERR", Err: errors.New("table does not exist")}}

	response := respond(t, results, nil)
	if response.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if response.Status != backend.StatusBadRequest {
		t.Errorf("expected status bad request, got %v", response.Status)
	}
	if len(response.Frames) != 0 {
		t.Errorf("expected no frames on error, got %d", len(response.Frames))
	}
}

func TestCreateDataResponse_QueryError(t *testing.T) {
	response := respond(t, nil, errors.New("connection refused"))
	if response.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if response.Status != backend.StatusBadRequest {
		t.Errorf("expected status bad request, got %v", response.Status)
	}
}
