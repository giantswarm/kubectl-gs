package release

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
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

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}
