package wcluster

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagsError = &microerror.Error{
	Kind: "invalidFlagsError",
}

var invalidBaseError = &microerror.Error{
	Kind: "invalidBaseError",
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var fetchFailedError = &microerror.Error{
	Kind: "fetchFailedError",
}
