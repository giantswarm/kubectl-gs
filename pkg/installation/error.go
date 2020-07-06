package installation

import (
	"github.com/giantswarm/microerror"
)

var cannotGetInstallationInfo = &microerror.Error{
	Kind: "cannotGetInstallationInfo",
}

// IsCannotGetInstallationInfo asserts cannotGetInstallationInfo.
func IsCannotGetInstallationInfo(err error) bool {
	return microerror.Cause(err) == cannotGetInstallationInfo
}
