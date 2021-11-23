package selfupdate

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var hasNewVersionError = &microerror.Error{
	Kind: "hasNewVersion",
}

// IsHasNewVersion asserts hasNewVersionError.
func IsHasNewVersion(err error) bool {
	return microerror.Cause(err) == hasNewVersionError
}

var versionNotFoundError = &microerror.Error{
	Kind: "versionNotFoundError",
}

// IsVersionNotFound asserts versionNotFoundError.
func IsVersionNotFound(err error) bool {
	return microerror.Cause(err) == versionNotFoundError
}
