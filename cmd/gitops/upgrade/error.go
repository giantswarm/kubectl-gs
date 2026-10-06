package upgrade

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// metadataNotFoundError is returned for repositories that hold no metadata
// file, and therefore no record of what to upgrade.
var metadataNotFoundError = &microerror.Error{
	Kind: "metadataNotFoundError",
}

// saveError is returned when the files were upgraded but the metadata file
// could not be written, leaving the two out of step.
var saveError = &microerror.Error{
	Kind: "saveError",
}

// dirtyTreeError is returned when the repository has uncommitted changes. A
// failed upgrade is undone with git, which would take those changes with it.
var dirtyTreeError = &microerror.Error{
	Kind: "dirtyTreeError",
}

// gitError is returned when git cannot tell whether the working tree is
// clean, typically because the repository is not a git repository.
var gitError = &microerror.Error{
	Kind: "gitError",
}

// upgradeFailedError is returned when a migration fails on disk.
var upgradeFailedError = &microerror.Error{
	Kind: "upgradeFailedError",
}

var invalidFlagsError = &microerror.Error{
	Kind: "invalidFlagsError",
}
