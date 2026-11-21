// Package migration brings the parts of a GitOps repository generated with an
// older repository structure up to the structure this kubectl-gs produces.
//
// Every bump of metadata.StructureVersion comes with a Migration that rewrites
// the affected layers from the previous version to the new one: adding,
// renaming or removing files within a layer, patching kustomizations.
// Migrations work on the
// files as they are on disk, rather than by re-rendering a layer, because the
// repository does not record the inputs a layer was generated from, and
// because customers are expected to edit the generated files by hand.
package migration

import (
	"fmt"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Migration moves the layers of a repository from structure version From to
// From+1.
type Migration struct {
	// From is the structure version the migration starts at.
	From int

	// Description says in one line what changed in the structure. It is
	// printed by `kubectl gs gitops upgrade`, so write it for the user.
	Description string

	// Kinds names the layer kinds the structure change affects. Layers of any
	// other kind are only bumped to From+1, with no files touched.
	Kinds []string

	// Apply rewrites one layer, found at layer.Path relative to repoPath, and
	// returns a human readable line per change made, which is printed to the
	// user. layer.StructureVersion is From.
	//
	// It runs against an in-memory copy of the repository, holding only its
	// directories and regular files, `.git` excluded, and then, unless it is
	// a dry run and when every step has succeeded there, against the
	// repository on disk. It must:
	//
	//   - work on the files as they are, as users edit generated files by
	//     hand, and fail rather than guess when a file is not as expected,
	//   - leave a layer that is already migrated as it is, so that it is safe
	//     to run twice,
	//   - not touch the repository metadata file, which the caller updates
	//     once every migration has run,
	//   - not change, move or delete files git ignores, such as decrypted
	//     secrets: a failed upgrade is undone with git, which cannot restore
	//     those,
	//   - not move the layer's own directory: steps are planned by the paths
	//     the layers had before the upgrade. A layer it moves inside its own
	//     directory, such as an app in a workload cluster, is followed with
	//     Relocate.
	//
	// Only layers are migrated. Files outside every layer, such as those
	// `gitops init` creates at the repository root, have no migration of
	// their own, so a structure change must not require changing them.
	Apply func(fs *afero.Afero, repoPath string, layer metadata.Layer, env Env) ([]string, error)

	// Relocate returns where a layer at the given path is after the
	// migration, for the layers it moves, of any kind. Nil when it moves
	// none.
	Relocate func(path string) string

	// ManualSteps says what the user has to do by hand once the migration
	// has run, for changes files alone can't make safely, such as those
	// Flux has to apply in a given order. Printed by `kubectl gs gitops
	// upgrade` when the migration ran, so write it for the user.
	ManualSteps string
}

// Env is what a migration knows about the repository besides its files.
type Env struct {
	// Ignored reports whether git ignores the file at the path, relative to
	// the repository root. Nil when that is not known, as outside git.
	Ignored func(path string) bool
}

// IsIgnored reports whether git ignores the file at the path.
func (e Env) IsIgnored(path string) bool {
	return e.Ignored != nil && e.Ignored(path)
}

// Step is one migration to run against one layer.
type Step struct {
	Layer     metadata.Layer
	Migration Migration
}

// Plan is the ordered list of steps that brings a repository to Target.
type Plan struct {
	Target int

	// Steps are ordered by structure version first and layer path second, so
	// that the whole repository is at version N before any layer moves to
	// N+1. A migration can therefore rely on its neighbours, e.g. a parent
	// kustomization, having already been migrated to the same version.
	Steps []Step

	// Layers are the layers the plan moves to Target, including those whose
	// kind no migration on the way affects.
	Layers []metadata.Layer

	// Moves maps the path of a layer a migration on the way relocates to
	// its path at Target.
	Moves map[string]string

	// Env is passed to every migration. The caller sets it before Run.
	Env Env
}

// Result is what running a plan did.
type Result struct {
	// Changes holds the lines each step reported, in the order they ran.
	Changes []StepChanges

	// Files lists the files a dry run would add, modify or remove. It is
	// empty for a real run: use git to see what that changed.
	Files []FileChange
}

// StepChanges are the changes a single step reported.
type StepChanges struct {
	Step    Step
	Changes []string
}

// StepError is returned when a migration fails. It keeps the migration's own
// error, so that callers can still inspect it, e.g. with
// errors.Is(err, fs.ErrNotExist).
type StepError struct {
	Step Step
	Err  error

	// OnDisk is whether the step failed on the repository, after it went
	// through on the in-memory copy, so files may have changed. A step that
	// fails on the copy fails before anything is written.
	OnDisk bool
}

func (e *StepError) Error() string {
	return fmt.Sprintf(
		"migrating %s %s from repository structure version %d to %d: %s",
		e.Step.Layer.Kind, e.Step.Layer.Path, e.Step.Migration.From, e.Step.Migration.From+1, e.Err,
	)
}

func (e *StepError) Unwrap() error {
	return e.Err
}
