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

var unknownUrlTypeError = &microerror.Error{
	Kind: "unknownUrlTypeError",
}

// IsUnknownUrlType asserts unknownUrlTypeError.
func IsUnknownUrlType(err error) bool {
	return microerror.Cause(err) == unknownUrlTypeError
}
