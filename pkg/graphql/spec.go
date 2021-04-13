package graphql

type Client interface {
	ExecuteQuery(query string, variables map[string]string, v interface{}) error
}
