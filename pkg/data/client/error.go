package client

import (
	"github.com/giantswarm/microerror"
)

var InvalidConfigError = &microerror.Error{
	Kind: "InvalidConfigError",
}

// IsInvalidConfig asserts InvalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == InvalidConfigError
}
