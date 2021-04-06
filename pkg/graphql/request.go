package graphql

type requestBody struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables,omitempty"`
}
