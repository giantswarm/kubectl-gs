package cluster

import (
	"github.com/giantswarm/microerror"
)

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

var insufficientPermissionsError = &microerror.Error{
	Kind: "insufficientPermissionsError",
}

// IsInsufficientPermissions asserts insufficientPermissionsError.
func IsInsufficientPermissions(err error) bool {
	return microerror.Cause(err) == insufficientPermissionsError
}
