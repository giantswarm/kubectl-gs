package nodepool

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

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

// IsInvalidFlag asserts invalidFlagError.
func IsInvalidFlag(err error) bool {
	return microerror.Cause(err) == invalidFlagError
}

var clusterNotFoundError = &microerror.Error{
	Kind: "clusterNotFoundError",
}

// IsClusterNotFound asserts clusterNotFoundError.
func IsClusterNotFound(err error) bool {
	return microerror.Cause(err) == clusterNotFoundError
}
