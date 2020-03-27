package appcatalog

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

// unmashalToMapFailedError is used when a YAML appCatalog values  can't be unmarshalled into map[string]interface{}.
var unmashalToMapFailedError = &microerror.Error{
	Kind: "unmashalToMapFailedError",
	Desc: "Could not unmarshal YAML into a map[string]interface{} structure. Seems like the YAML is invalid.",
}

// IsUnmashalToMapFailed asserts unmashalToMapFailedError.
func IsUnmashalToMapFailed(err error) bool {
	return microerror.Cause(err) == unmashalToMapFailedError
}
