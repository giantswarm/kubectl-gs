package helmbinary

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

var commandError = &microerror.Error{
	Kind: "commandError",
}

// IsCommand asserts CommandError.
func IsCommand(err error) bool {
	return microerror.Cause(err) == commandError
}
