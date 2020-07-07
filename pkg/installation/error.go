package installation

import (
	"github.com/giantswarm/microerror"
)

var cannotParseCertificateError = &microerror.Error{
	Kind: "cannotParseCertificateError",
}

// IsCannotParseCertificate asserts cannotParseCertificateError.
func IsCannotParseCertificate(err error) bool {
	return microerror.Cause(err) == cannotParseCertificateError
}

var cannotGetInstallationInfo = &microerror.Error{
	Kind: "cannotGetInstallationInfo",
}

// IsCannotGetInstallationInfo asserts cannotGetInstallationInfo.
func IsCannotGetInstallationInfo(err error) bool {
	return microerror.Cause(err) == cannotGetInstallationInfo
}

var unknownUrlTypeError = &microerror.Error{
	Kind: "unknownUrlTypeError",
}

// IsUnknownUrlType asserts unknownUrlTypeError.
func IsUnknownUrlType(err error) bool {
	return microerror.Cause(err) == unknownUrlTypeError
}
