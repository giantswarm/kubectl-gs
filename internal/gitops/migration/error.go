package migration

import (
	"errors"

	"github.com/giantswarm/microerror"
)

// missingMigrationError is returned when no migration is registered for a
// structure version the repository has to go through. It means
// metadata.StructureVersion was bumped without adding a migration, which the
// registry test is there to catch.
var missingMigrationError = &microerror.Error{
	Kind: "missingMigrationError",
}

// unversionedLayerError is returned for layers recorded at structure version
// 0, which predates structure versioning. There is no migration from there, as
// nothing tells what such a layer looks like.
var unversionedLayerError = &microerror.Error{
	Kind: "unversionedLayerError",
}

// IsUnversionedLayer asserts unversionedLayerError.
func IsUnversionedLayer(err error) bool {
	return microerror.Cause(err) == unversionedLayerError
}

// unversionedRepositoryError is returned when the repository level record
// holds no structure version.
var unversionedRepositoryError = &microerror.Error{
	Kind: "unversionedRepositoryError",
}

// newerLayerError is returned when the repository, or one of its layers, is
// recorded at a newer structure version than the target, i.e. was touched by
// a newer kubectl-gs than the one running.
var newerLayerError = &microerror.Error{
	Kind: "newerLayerError",
}

// IsNewerLayer asserts newerLayerError.
func IsNewerLayer(err error) bool {
	return microerror.Cause(err) == newerLayerError
}

// IsMigrationFailed asserts that a migration's Apply failed, see StepError.
func IsMigrationFailed(err error) bool {
	var se *StepError
	return errors.As(err, &se)
}

// stagingError is returned when the repository cannot be read into the
// in-memory copy a dry run works on.
var stagingError = &microerror.Error{
	Kind: "stagingError",
}
