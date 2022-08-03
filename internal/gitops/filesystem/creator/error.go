package creator

import (
	"github.com/giantswarm/microerror"
)

var validationError = &microerror.Error{
	Kind: "validationError",
}

// IsValidationError asserts validationError.
func IsValidationError(err error) bool {
	return microerror.Cause(err) == validationError
}
