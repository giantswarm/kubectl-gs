package credentialplugin

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var credentialPluginError = &microerror.Error{
	Kind: "credentialPluginError",
}

// IsCredentialPluginError asserts credentialPluginError.
func IsCredentialPluginError(err error) bool {
	return microerror.Cause(err) == credentialPluginError
}
