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
	// It runs against the repository on disk, or, for a dry run, against an
	// in-memory copy holding only its directories and regular files, `.git`
	// excluded. It must:
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
	//   - not move the layer's own directory: steps are planned, and the
	//     upgraded layers recorded, by the paths the layers had before the
	//     upgrade. A structure change that moves layers needs the planner
	//     taught to follow them first.
	//
	// Only layers are migrated. Files outside every layer, such as those
	// `gitops init` creates at the repository root, have no migration of
	// their own, so a structure change must not require changing them.
	Apply func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error)
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
