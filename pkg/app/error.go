package app

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidTypeError = &microerror.Error{
	Kind: "invalidTypeError",
}

// IsInvalidType asserts invalidTypeError.
func IsInvalidType(err error) bool {
	return microerror.Cause(err) == invalidTypeError
}

var noResourcesError = &microerror.Error{
	Kind: "noResourcesError",
}

// IsNoResources asserts noResourcesError.
func IsNoResources(err error) bool {
	return microerror.Cause(err) == noResourcesError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var noSchemaError = &microerror.Error{
	Kind: "noSchemaError",
}

// IsNoSchema asserts noSchemaError.
func IsNoSchema(err error) bool {
	return microerror.Cause(err) == noSchemaError
}

var fetchError = &microerror.Error{
	Kind: "fetchError",
}

// IsFetch asserts fetchError.
func IsFetch(err error) bool {
	return microerror.Cause(err) == fetchError
}

var commandError = &microerror.Error{
	Kind: "commandError",
}

// IsCommand asserts CommandError.
func IsCommand(err error) bool {
	return microerror.Cause(err) == commandError
}

var ioError = &microerror.Error{
	Kind: "ioError",
}

// IsIO asserts ioError.
func IsIO(err error) bool {
	return microerror.Cause(err) == ioError
}

var otherError = &microerror.Error{
	Kind: "otherError",
}

// IsOther asserts otherError.
func IsOther(err error) bool {
	return microerror.Cause(err) == otherError
}
