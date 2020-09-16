package provider

import "github.com/giantswarm/microerror"

var invalidObjectDefinitionError = &microerror.Error{
	Kind: "invalidObjectDefinitionError",
}

// IsInvalidObjectDefinition asserts invalidObjectDefinitionError.
func IsInvalidObjectDefinition(err error) bool {
	return microerror.Cause(err) == invalidObjectDefinitionError
}
