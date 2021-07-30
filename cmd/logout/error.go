package logout

import "github.com/giantswarm/microerror"

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

var noKubeconfig = &microerror.Error{
	Kind: "noKubeconfig",
	Desc: "no kubeconfig found",
}

// IsNoKubeconfig asserts noKubeconfig.
func IsNoKubeconfig(err error) bool {
	return microerror.Cause(err) == noKubeconfig
}

var noKubeconfigCurrentContext = &microerror.Error{
	Kind: "noKubeconfigCurrentContext",
	Desc: "no current kubeconfig context set",
}

// IsNoKubeconfigCurrentContext asserts noKubeconfigCurrentContext.
func IsNoKubeconfigCurrentContext(err error) bool {
	return microerror.Cause(err) == noKubeconfigCurrentContext
}

var noKubeconfigCurrentUser = &microerror.Error{
	Kind: "noKubeconfigCurrentUser",
	Desc: "no current kubeconfig user set",
}

// IsNoKubeconfigCurrentContext asserts noKubeconfigCurrentUser.
func IsNoKubeconfigCurrentUser(err error) bool {
	return microerror.Cause(err) == noKubeconfigCurrentUser
}

var noIDToken = &microerror.Error{
	Kind: "noIDToken",
	Desc: "no ID token present",
}

// IsNoIDToken asserts noIDToken.
func IsNoIDToken(err error) bool {
	return microerror.Cause(err) == noIDToken
}

var noRefreshToken = &microerror.Error{
	Kind: "noRefreshToken",
	Desc: "no refresh token present",
}

// IsNoRefreshToken asserts noRefreshToken.
func IsNoRefreshToken(err error) bool {
	return microerror.Cause(err) == noRefreshToken
}
