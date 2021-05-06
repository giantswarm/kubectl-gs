package graphql

import (
	"errors"
	"strings"

	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var unknownAuthorityError = &microerror.Error{
	Kind: "unknownAuthorityError",
	Desc: "the server side has presented a certificate that was issued by an untrusted authority",
}

// IsUnknownAuthority asserts unknownAuthorityError.
func IsUnknownAuthority(err error) bool {
	return microerror.Cause(err) == unknownAuthorityError
}

var httpError = &microerror.Error{
	Kind: "httpError",
}

// IsHttp asserts httpError.
func IsHttp(err error) bool {
	return microerror.Cause(err) == httpError
}

var queryError = &microerror.Error{
	Kind: "queryError",
}

// IsQuery asserts queryError.
func IsQuery(err error) bool {
	return microerror.Cause(err) == queryError
}

type ResponseErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ResponseError as defined per GraphQL spec.
//
// https://spec.graphql.org/June2018/#sec-Errors
type ResponseError struct {
	Message    string                  `json:"message"`
	Locations  []ResponseErrorLocation `json:"locations"`
	Path       []interface{}           `json:"path"`
	Extensions map[string]interface{}  `json:"extensions"`
}

type ResponseErrorCollection []ResponseError

func (r ResponseErrorCollection) Error() string {
	var builder strings.Builder

	for _, error := range r {
		if builder.Len() > 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(error.Message)
	}

	return builder.String()
}

func IsResponseErrorCollection(err error) bool {
	responseErrorCollection := &ResponseErrorCollection{}

	return errors.As(err, responseErrorCollection)
}
