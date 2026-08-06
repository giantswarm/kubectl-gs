package check

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// outOfSyncError is returned when the repository was generated with an older
// structure version than the one this kubectl-gs produces. It carries no
// detail of its own: the report is printed to stdout first, and this only
// makes the command exit non-zero so it is usable as a CI check.
var outOfSyncError = &microerror.Error{
	Kind: "outOfSyncError",
}

// metadataNotFoundError is returned for repositories created before
// kubectl-gs started recording the structure version, which is the normal
// case for every repository that predates this feature.
var metadataNotFoundError = &microerror.Error{
	Kind: "metadataNotFoundError",
}
