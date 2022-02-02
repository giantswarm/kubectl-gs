package key

import (
	"github.com/giantswarm/microerror"
)

// clusterIDInvalidError is used to check ClusterID correctness
var clusterIDInvalidError = &microerror.Error{
	Kind: "clusterIDInvalidError",
}

// IsClusterIDInvalid asserts clusterIDInvalidError.
func IsClusterIDInvalid(err error) bool {
	return microerror.Cause(err) == clusterIDInvalidError
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

// parsingReleaseError is used when trying to parse the Release name as a SemVer fails.
var parsingReleaseError = &microerror.Error{
	Kind: "parsingReleaseError",
	Desc: "Could not parse the Release as a SemVer.",
}

// IsUnmashalToMapFailed asserts parsingReleaseError.
func IsParsingReleaseError(err error) bool {
	return microerror.Cause(err) == unmashalToMapFailedError
}
