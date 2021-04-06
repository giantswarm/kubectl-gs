package graphql

import "encoding/json"

type Client interface {
	ExecuteQuery(query string, variables map[string]string) (*json.RawMessage, error)
}
