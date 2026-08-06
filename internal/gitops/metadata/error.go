package metadata

import (
	"github.com/giantswarm/microerror"
)

// NotFoundError is returned by Load when the repository holds no metadata file
// yet. Callers are expected to tell these repositories apart from broken ones,
// because every repository created before this feature existed hits this case.
var NotFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts NotFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == NotFoundError
}

var invalidMetadataError = &microerror.Error{
	Kind: "invalidMetadataError",
}
