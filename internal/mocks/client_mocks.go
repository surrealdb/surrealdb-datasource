package mocks

import (
	"context"

	"github.com/surrealdb/surrealdb-datasource/pkg/client"
)

// MockSurrealDBClient is a mock implementation of the client.SurrealDBClient
// interface. Each method delegates to its corresponding func field, which tests
// set as needed.
type MockSurrealDBClient struct {
	SignInFunc func(ctx context.Context, config *client.SurrealConfig) (string, error)
	UseFunc    func(ctx context.Context, namespace, database string) error
	QueryFunc  func(ctx context.Context, sql string, vars map[string]any) ([]client.QueryResult, error)
	CloseFunc  func(ctx context.Context) error
}

func (m *MockSurrealDBClient) SignIn(ctx context.Context, config *client.SurrealConfig) (string, error) {
	return m.SignInFunc(ctx, config)
}

func (m *MockSurrealDBClient) Use(ctx context.Context, namespace, database string) error {
	return m.UseFunc(ctx, namespace, database)
}

func (m *MockSurrealDBClient) Query(ctx context.Context, sql string, vars map[string]any) ([]client.QueryResult, error) {
	return m.QueryFunc(ctx, sql, vars)
}

func (m *MockSurrealDBClient) Close(ctx context.Context) error {
	return m.CloseFunc(ctx)
}
