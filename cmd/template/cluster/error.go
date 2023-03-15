package cluster

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

var templateFlagNotImplemented = &microerror.Error{
	Kind: "templateFlagsNotImplementedError",
}

// IsTemplateFlagNotImplemented asserts templateFlagNotImplemented
func IsTemplateFlagNotImplemented(err error) bool {
	return microerror.Cause(err) == templateFlagNotImplemented
}
