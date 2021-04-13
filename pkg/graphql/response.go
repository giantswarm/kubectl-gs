package graphql

import "encoding/json"

type responseBody struct {
	Data   *json.RawMessage        `json:"data"`
	Errors ResponseErrorCollection `json:"errors"`
}
