package release

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

var noResourcesError = &microerror.Error{
	Kind: "noResourcesError",
}

// IsNoResources asserts noResourcesError.
func IsNoResources(err error) bool {
	return microerror.Cause(err) == noResourcesError
}

var noMatchError = &microerror.Error{
	Kind: "noMatchError",
}

// IsNoMatch asserts noMatchError.
func IsNoMatch(err error) bool {
	return microerror.Cause(err) == noMatchError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var invalidProviderError = &microerror.Error{
	Kind: "invalidProviderError",
}

// IsInvalidProvider asserts invalidProviderError.
func IsInvalidProvider(err error) bool {
	return microerror.Cause(err) == invalidProviderError
}
