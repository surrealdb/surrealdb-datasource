package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/grafana-labs/surrealdb-datasource/pkg/client"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// CreateDataResponse runs a single data query against SurrealDB and converts the
// result into a Grafana data response.
func (d *SurrealDatasource) CreateDataResponse(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	str, err := sqlStringFromDataQuery(query)
	if err != nil {
		return backend.ErrDataResponseWithSource(backend.StatusBadRequest, backend.ErrorSourcePlugin, fmt.Sprintf("sql: %v", err.Error()))
	}

	results, err := d.client.Query(ctx, str, nil)
	if err != nil {
		return backend.ErrDataResponseWithSource(backend.StatusBadRequest, backend.ErrorSourceDownstream, fmt.Sprintf("query: %v", err.Error()))
	}

	return buildResponse(results)
}

// sqlStringFromDataQuery converts a data query into a SurrealQL string,
// interpolating any macros (including the SurrealDB-specific time macros).
func sqlStringFromDataQuery(query backend.DataQuery) (string, error) {
	sq, err := sqlutil.GetQuery(query)
	if err != nil {
		return "", err
	}

	str, err := sqlutil.Interpolate(sq, macros)
	if err != nil {
		return "", err
	}

	return str, nil
}

// buildResponse converts the per-statement results from SurrealDB into a data
// response. Each statement that returns rows becomes its own frame; the first
// statement that reports an error short-circuits into an error response.
func buildResponse(results []client.QueryResult) backend.DataResponse {
	var response backend.DataResponse

	for i, result := range results {
		if result.Err != nil || result.Status == statusErr {
			msg := result.Status
			if result.Err != nil {
				msg = result.Err.Error()
			}
			return backend.ErrDataResponseWithSource(backend.StatusBadRequest, backend.ErrorSourceDownstream, fmt.Sprintf("query: %v", msg))
		}

		if len(result.Rows) == 0 {
			continue
		}

		name := "response"
		if i > 0 {
			name = fmt.Sprintf("response_%d", i)
		}
		response.Frames = append(response.Frames, toDataFrame(name, result.Rows))
	}

	return response
}

// statusErr is the status SurrealDB reports for a failed statement.
const statusErr = "ERR"

// toDataFrame builds a typed Grafana data frame from a set of SurrealDB rows.
// Columns are inferred from the union of keys across all rows and sorted by name
// for stable output. Each column is given the narrowest Grafana-friendly type
// that fits all of its values, using nullable fields so missing values and NONE
// become null rather than breaking the frame.
func toDataFrame(name string, rows []map[string]any) *data.Frame {
	frame := data.NewFrame(name)

	if len(rows) == 0 {
		return frame
	}

	columns := map[string]struct{}{}
	for _, row := range rows {
		for key := range row {
			columns[key] = struct{}{}
		}
	}

	keys := make([]string, 0, len(columns))
	for key := range columns {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		frame.Fields = append(frame.Fields, fieldForColumn(key, rows))
	}

	return frame
}

type columnKind int

const (
	kindString columnKind = iota
	kindTime
	kindInt
	kindFloat
	kindBool
)

// fieldForColumn picks a column type that fits every value in the column and
// builds the corresponding nullable field.
func fieldForColumn(key string, rows []map[string]any) *data.Field {
	switch columnKindFor(key, rows) {
	case kindTime:
		values := make([]*time.Time, len(rows))
		for i, row := range rows {
			values[i] = asTime(row[key])
		}
		return data.NewField(key, nil, values)
	case kindInt:
		values := make([]*int64, len(rows))
		for i, row := range rows {
			values[i] = asInt(row[key])
		}
		return data.NewField(key, nil, values)
	case kindFloat:
		values := make([]*float64, len(rows))
		for i, row := range rows {
			values[i] = asFloat(row[key])
		}
		return data.NewField(key, nil, values)
	case kindBool:
		values := make([]*bool, len(rows))
		for i, row := range rows {
			values[i] = asBool(row[key])
		}
		return data.NewField(key, nil, values)
	default:
		values := make([]*string, len(rows))
		for i, row := range rows {
			values[i] = asString(row[key])
		}
		return data.NewField(key, nil, values)
	}
}

// columnKindFor determines the widest column kind that all non-null values in a
// column are compatible with. Any mix of incompatible categories falls back to
// string, which can represent anything.
func columnKindFor(key string, rows []map[string]any) columnKind {
	var sawNonNil, sawTime, sawInt, sawFloat, sawBool, sawOther bool

	for _, row := range rows {
		value, ok := row[key]
		if !ok || value == nil {
			continue
		}
		sawNonNil = true

		switch value.(type) {
		case models.CustomDateTime, *models.CustomDateTime, time.Time, *time.Time:
			sawTime = true
		case bool:
			sawBool = true
		case float32, float64:
			sawFloat = true
		case int, int64, uint64:
			sawInt = true
		default:
			sawOther = true
		}
	}

	switch {
	case !sawNonNil:
		return kindString
	case sawOther:
		return kindString
	case sawTime && !sawInt && !sawFloat && !sawBool:
		return kindTime
	case sawBool && !sawInt && !sawFloat && !sawTime:
		return kindBool
	case (sawInt || sawFloat) && !sawTime && !sawBool:
		if sawFloat {
			return kindFloat
		}
		return kindInt
	default:
		return kindString
	}
}

func asTime(value any) *time.Time {
	switch v := value.(type) {
	case models.CustomDateTime:
		t := v.Time.UTC()
		return &t
	case *models.CustomDateTime:
		if v == nil {
			return nil
		}
		t := v.Time.UTC()
		return &t
	case time.Time:
		t := v.UTC()
		return &t
	case *time.Time:
		if v == nil {
			return nil
		}
		t := v.UTC()
		return &t
	default:
		return nil
	}
}

// asInt handles the integer types the CBOR decoder produces for SurrealDB
// numbers (uint64 for non-negative, int64 for negative), plus plain int for
// values constructed in Go.
func asInt(value any) *int64 {
	switch v := value.(type) {
	case int:
		n := int64(v)
		return &n
	case int64:
		return &v
	case uint64:
		n := int64(v)
		return &n
	default:
		return nil
	}
}

func asFloat(value any) *float64 {
	switch v := value.(type) {
	case float32:
		f := float64(v)
		return &f
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case uint64:
		f := float64(v)
		return &f
	default:
		return nil
	}
}

func asBool(value any) *bool {
	if v, ok := value.(bool); ok {
		return &v
	}
	return nil
}

// asString renders any SurrealDB value as a string. Strings pass through;
// record IDs, durations and other SurrealDB model types use their canonical
// representation; everything else (nested objects, arrays, geometry) is encoded
// as JSON so no data is lost.
func asString(value any) *string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return &v
	case models.RecordID:
		s := v.String()
		return &s
	case *models.RecordID:
		if v == nil {
			return nil
		}
		s := v.String()
		return &s
	case models.Table:
		s := string(v)
		return &s
	case models.CustomDuration:
		s := v.String()
		return &s
	case models.CustomDurationString:
		s := string(v)
		return &s
	case fmt.Stringer:
		s := v.String()
		return &s
	default:
		b, err := json.Marshal(v)
		if err != nil {
			s := fmt.Sprintf("%v", v)
			return &s
		}
		s := string(b)
		return &s
	}
}
