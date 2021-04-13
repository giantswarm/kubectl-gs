package graphql

import "context"

type Client interface {
	ExecuteQuery(ctx context.Context, query string, variables map[string]string, v interface{}) error
}
