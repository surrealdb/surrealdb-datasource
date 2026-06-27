package client

import (
	"context"

	surrealdb "github.com/surrealdb/surrealdb.go"
)

// SurrealConfig defines the configuration for the SurrealDB database.
type SurrealConfig struct {
	Database  string `json:"database,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Password  string `json:"password,omitempty"`
	Username  string `json:"username,omitempty"`
	// Access is the SurrealDB access method used for record-level authentication.
	Access string `json:"access,omitempty"`
	// AuthScope selects the level a system user authenticates at: "root"
	// (default), "namespace", or "database". SurrealDB infers the level from the
	// fields present in the sign-in payload, so this controls whether the
	// namespace and database are included. Ignored when Access is set.
	AuthScope string `json:"authScope,omitempty"`
}

// Authentication levels for a SurrealDB system user. They map to the fields
// included in the sign-in payload: root sends neither namespace nor database,
// namespace sends the namespace, and database sends both.
const (
	AuthScopeRoot      = "root"
	AuthScopeNamespace = "namespace"
	AuthScopeDatabase  = "database"
)

// QueryResult is the plugin-owned representation of a single SurrealQL
// statement result. It mirrors surrealdb.QueryResult but decouples the rest of
// the plugin (and its tests/mocks) from the SDK's generic types.
type QueryResult struct {
	// Status is "OK" for a successful statement or "ERR" for a failed one.
	Status string
	// Time is the server-reported execution time for the statement.
	Time string
	// Rows holds the records returned by the statement.
	Rows []map[string]any
	// Err is set when the statement failed (Status == "ERR").
	Err error
}

// SurrealDBClient is the minimal surface of the SurrealDB SDK that the
// datasource depends on. Keeping it small makes it trivial to mock in tests:
// the SDK's real query function is a package-level generic and cannot be placed
// on an interface, so realClient adapts it behind these decoded-result methods.
type SurrealDBClient interface {
	SignIn(ctx context.Context, config *SurrealConfig) (string, error)
	Use(ctx context.Context, namespace, database string) error
	Query(ctx context.Context, sql string, vars map[string]any) ([]QueryResult, error)
	Close(ctx context.Context) error
}

// Client wraps a SurrealDBClient and orchestrates the connection lifecycle.
type Client struct {
	db SurrealDBClient
}

// Use returns a new Client backed by the given SurrealDBClient.
func Use(db SurrealDBClient) *Client {
	return &Client{db: db}
}

// Dial opens a connection to the SurrealDB endpoint. The returned client is not
// yet authenticated; call Client.Connect to sign in and select the namespace
// and database.
func Dial(ctx context.Context, endpoint string) (SurrealDBClient, error) {
	db, err := surrealdb.FromEndpointURLString(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &realClient{db: db}, nil
}

// Connect selects the configured namespace and database, then authenticates.
//
// Use is called before SignIn because the HTTP transport is stateless and must
// have the namespace/database set to build the request headers for every RPC,
// including sign-in itself; otherwise it fails with "namespace or database or
// both are not set". On WebSocket the order is immaterial.
func (c *Client) Connect(ctx context.Context, config *SurrealConfig) error {
	if err := c.db.Use(ctx, config.Namespace, config.Database); err != nil {
		return err
	}

	if _, err := c.db.SignIn(ctx, config); err != nil {
		return err
	}

	return nil
}

// Query runs a SurrealQL statement (or statements) and returns one QueryResult
// per statement.
func (c *Client) Query(ctx context.Context, sql string, vars map[string]any) ([]QueryResult, error) {
	return c.db.Query(ctx, sql, vars)
}

// Close releases the underlying connection.
func (c *Client) Close(ctx context.Context) error {
	return c.db.Close(ctx)
}

// realClient adapts the SurrealDB SDK (*surrealdb.DB) to the SurrealDBClient
// interface.
type realClient struct {
	db *surrealdb.DB
}

// buildAuth maps the datasource configuration onto the SDK auth payload.
//
// SurrealDB infers the authentication level from the fields present in the
// sign-in payload, so the namespace/database are included only as needed:
//   - record access (Access set): namespace + database + access method
//   - database-level user: namespace + database
//   - namespace-level user: namespace
//   - root-level user (default): neither
//
// The configured namespace/database are always applied separately via Use to
// select the working context, regardless of the sign-in level.
func buildAuth(config *SurrealConfig) surrealdb.Auth {
	auth := surrealdb.Auth{
		Username: config.Username,
		Password: config.Password,
	}

	if config.Access != "" {
		auth.Namespace = config.Namespace
		auth.Database = config.Database
		auth.Access = config.Access
		return auth
	}

	switch config.AuthScope {
	case AuthScopeNamespace:
		auth.Namespace = config.Namespace
	case AuthScopeDatabase:
		auth.Namespace = config.Namespace
		auth.Database = config.Database
	}

	return auth
}

func (r *realClient) SignIn(ctx context.Context, config *SurrealConfig) (string, error) {
	return r.db.SignIn(ctx, buildAuth(config))
}

func (r *realClient) Use(ctx context.Context, namespace, database string) error {
	return r.db.Use(ctx, namespace, database)
}

func (r *realClient) Close(ctx context.Context) error {
	return r.db.Close(ctx)
}

func (r *realClient) Query(ctx context.Context, sql string, vars map[string]any) ([]QueryResult, error) {
	// Query into `any` rather than []map[string]any: SurrealDB statements can
	// return a single object, a scalar (e.g. RETURN 1, used by the health check),
	// or an array of scalars, none of which unmarshal into []map[string]any.
	// normalizeRows coerces each statement's result into table rows.
	//
	// surrealdb.Query returns one result per statement. When a statement fails it
	// sets the per-statement Error and also returns a joined top-level error; the
	// results slice is still populated, so we keep it and let the caller inspect
	// each statement's Status/Err.
	res, err := surrealdb.Query[any](ctx, r.db, sql, vars)
	if res == nil {
		return nil, err
	}

	results := make([]QueryResult, 0, len(*res))
	for _, qr := range *res {
		result := QueryResult{
			Status: qr.Status,
			Time:   qr.Time,
			Rows:   normalizeRows(qr.Result),
		}
		// Avoid the typed-nil interface pitfall: only assign when non-nil.
		if qr.Error != nil {
			result.Err = qr.Error
		}
		results = append(results, result)
	}

	return results, nil
}

// normalizeRows coerces a single statement's result into table rows. SurrealDB
// statements can return an array of objects (the common case), a single object,
// an array of scalars, or a bare scalar; the non-object shapes are wrapped into
// a single "value" column so every result can be rendered as a frame.
func normalizeRows(result any) []map[string]any {
	switch v := result.(type) {
	case nil:
		return nil
	case []map[string]any:
		return v
	case []any:
		rows := make([]map[string]any, 0, len(v))
		for _, element := range v {
			if m, ok := element.(map[string]any); ok {
				rows = append(rows, m)
			} else {
				rows = append(rows, map[string]any{"value": element})
			}
		}
		return rows
	case map[string]any:
		return []map[string]any{v}
	default:
		return []map[string]any{{"value": v}}
	}
}
