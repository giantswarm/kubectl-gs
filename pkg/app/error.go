package app

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidTypeError = &microerror.Error{
	Kind: "invalidTypeError",
}

var noResourcesError = &microerror.Error{
	Kind: "noResourcesError",
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
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

var ioError = &microerror.Error{
	Kind: "ioError",
}
