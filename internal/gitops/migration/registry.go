package migration

// Migrations holds one migration per structure version bump, keyed by the
// version it starts at.
//
// When bumping metadata.StructureVersion, add the migration from the previous
// version here, and record what changed in the gitops-template CHANGELOG.
var Migrations = []Migration{
	helmReleases,
}
