package installation

import (
	"github.com/giantswarm/microerror"
)

var cannotGetInstallationInfoError = &microerror.Error{
	Kind: "cannotGetInstallationInfoError",
}

// IsCannotGetInstallationInfo asserts cannotGetInstallationInfoError.
func IsCannotGetInstallationInfo(err error) bool {
	return microerror.Cause(err) == cannotGetInstallationInfoError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
