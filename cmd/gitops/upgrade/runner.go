package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/migration"
)

type runner struct {
	flag   *flag
	fs     *afero.Afero
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer

	// dirtyPaths, migrations and target default to asking git, the
	// registered migrations and the structure version this kubectl-gs
	// produces. Tests override them, as there is nothing to migrate at
	// structure version 1.
	dirtyPaths func(repoPath string) ([]string, error)
	migrations []migration.Migration
	target     int

	// tracked records whether git tracks the repository, as found by
	// ensureClean.
	tracked bool
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	repoPath := "."
	if f := cmd.InheritedFlags().Lookup("local-path"); f != nil {
		repoPath = f.Value.String()
	}

	dryRun := false
	if f := cmd.InheritedFlags().Lookup("dry-run"); f != nil {
		dryRun, _ = strconv.ParseBool(f.Value.String())
	}

	// Walking a repository path that is itself a symlink would not descend
	// into it, so resolve that one case up front. Paths merely below a
	// symlinked directory walk fine, and are left as the user wrote them.
	if _, ok := r.fs.Fs.(*afero.OsFs); ok {
		info, err := os.Lstat(repoPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(repoPath)
			if err == nil {
				repoPath = resolved
			}
		}
	}

	exists, err := r.fs.Exists(repoPath)
	if err != nil {
		return microerror.Mask(err)
	}
	if !exists {
		return microerror.Maskf(invalidFlagsError, "--local-path %q does not exist", repoPath)
	}
	isDir, err := r.fs.DirExists(repoPath)
	if err != nil {
		return microerror.Mask(err)
	}
	if !isDir {
		return microerror.Maskf(invalidFlagsError, "--local-path %q is not a directory", repoPath)
	}

	md, err := metadata.Load(r.fs, repoPath)
	if metadata.IsNotFound(err) {
		return microerror.Maskf(
			metadataNotFoundError,
			"%s holds no %s, so there is no record of what to upgrade.\nRun `kubectl gs gitops check --adopt` to record what the repository already contains.",
			repoPath,
			metadata.FileName,
		)
	} else if err != nil {
		return microerror.Mask(err)
	}

	plan, err := migration.NewPlan(md, r.migrations, r.target)
	if err != nil {
		return microerror.Mask(err)
	}

	if plan.Empty() && md.StructureVersion >= r.target {
		_, _ = fmt.Fprintf(r.stdout, "The repository is already at repository structure version %d, nothing to upgrade.\n", r.target)
		return nil
	}

	// Only migrations change files other than the metadata file, and only a
	// real run can leave them half changed.
	guarded := !dryRun && len(plan.Steps) > 0
	if guarded {
		err = r.ensureClean(repoPath)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if !dryRun {
		// Nothing to guard, only whether to suggest `git diff` afterwards.
		_, err = r.dirtyPaths(repoPath)
		r.tracked = err == nil
	}

	result, err := plan.Run(r.fs, repoPath, md, dryRun)
	if migration.IsMigrationFailed(err) {
		if !guarded {
			return microerror.Maskf(upgradeFailedError, "%s\nNothing was written to the repository.", err)
		}
		return microerror.Maskf(upgradeFailedError, "%s\n%s", err, r.undoHint(repoPath))
	} else if err != nil {
		return microerror.Mask(err)
	}

	if dryRun {
		r.printResult(plan, result, true)

		body, err := metadata.Render(md)
		if err != nil {
			return microerror.Mask(err)
		}

		_, _ = fmt.Fprintf(r.stdout, "\n%s\n%s\n", metadata.FilePath(repoPath), string(body))
		return nil
	}

	err = metadata.Save(r.fs, repoPath, md)
	if err != nil {
		if !guarded {
			return microerror.Maskf(saveError, "recording the upgrade in %s failed: %s", metadata.FilePath(repoPath), err)
		}
		return microerror.Maskf(
			saveError,
			"the files were upgraded, but recording that in %s failed: %s\n%s",
			metadata.FilePath(repoPath),
			err,
			r.undoHint(repoPath),
		)
	}

	r.printResult(plan, result, false)

	_, _ = fmt.Fprintf(r.stdout, "\nWrote %s\n", metadata.FilePath(repoPath))
	if r.tracked {
		_, _ = fmt.Fprintf(r.stdout, "\nReview the changes with `%s` before committing them.\n", gitCommand(repoPath, "diff"))
	} else {
		_, _ = fmt.Fprintf(r.stdout, "\nReview the changes before committing them.\n")
	}

	return nil
}

// ensureClean refuses to upgrade a repository with uncommitted changes, or one
// git does not track, as the only way to undo a failed upgrade is to reset the
// working tree with git. With --force it goes ahead, but remembers whether git
// can still help, for undoHint.
func (r *runner) ensureClean(repoPath string) error {
	dirty, err := r.dirtyPaths(repoPath)
	r.tracked = err == nil

	if r.flag.Force {
		return nil
	}

	switch {
	case errors.Is(err, errGitNotFound):
		return microerror.Maskf(
			gitError,
			"git is not installed. The upgrade relies on git to undo a failed run: install git, or pass --%s to upgrade anyway.",
			flagForce,
		)
	case errors.Is(err, errNotTracked):
		return microerror.Maskf(
			gitError,
			"git tracks nothing under %s: it is not in a git repository, or is ignored by the one it is in. The upgrade relies on git to undo a failed run: commit the repository first, or pass --%s to upgrade anyway.",
			repoPath,
			flagForce,
		)
	case err != nil:
		return microerror.Maskf(gitError, "%s\nPass --%s to upgrade anyway.", err, flagForce)
	}

	if len(dirty) == 0 {
		return nil
	}

	const shown = 10
	list := dirty
	more := ""
	if len(list) > shown {
		more = fmt.Sprintf("\n  ... and %d more", len(list)-shown)
		list = list[:shown]
	}

	return microerror.Maskf(
		dirtyTreeError,
		"%s has uncommitted changes:\n  %s%s\nCommit or stash them first, so that a failed upgrade can be undone without losing them, or pass --%s to upgrade anyway.",
		repoPath,
		strings.Join(list, "\n  "),
		more,
		flagForce,
	)
}

// undoHint tells the user how to get back to where the upgrade started. With
// --force, the working tree may not have been clean, so resetting it is not
// safe to suggest, and outside git there is nothing to suggest at all.
func (r *runner) undoHint(repoPath string) string {
	switch {
	case !r.tracked:
		return "Some files may already be changed, and as git does not track the repository, they cannot be restored automatically."
	case r.flag.Force:
		return fmt.Sprintf("Some files may already be changed. Check them with `%s` before running the upgrade again.", gitCommand(repoPath, "status"))
	default:
		return fmt.Sprintf("Some files may already be changed. Undo them with `%s` before running the upgrade again.", restoreHint(repoPath))
	}
}

func (r *runner) printResult(plan *migration.Plan, result *migration.Result, dryRun bool) {
	verb := "Upgraded"
	if dryRun {
		verb = "Would upgrade"
	}

	if plan.Empty() {
		// Only the repository level record is behind, no layer is.
		_, _ = fmt.Fprintf(r.stdout, "%s the repository record to repository structure version %d.\n", verb, plan.Target)
		return
	}

	_, _ = fmt.Fprintf(r.stdout, "%s %d layer(s) to repository structure version %d.\n", verb, len(plan.Layers), plan.Target)

	migrated := map[string]bool{}
	from := 0
	for _, sc := range result.Changes {
		migrated[sc.Step.Layer.Kind+"/"+sc.Step.Layer.Path] = true

		if sc.Step.Migration.From != from {
			from = sc.Step.Migration.From
			_, _ = fmt.Fprintf(r.stdout, "\n%d -> %d: %s\n", from, from+1, sc.Step.Migration.Description)
		}

		_, _ = fmt.Fprintf(r.stdout, "  %s %s\n", sc.Step.Layer.Kind, sc.Step.Layer.Path)
		for _, c := range sc.Changes {
			_, _ = fmt.Fprintf(r.stdout, "    - %s\n", c)
		}
	}

	var untouched []string
	for _, l := range plan.Layers {
		if !migrated[l.Kind+"/"+l.Path] {
			untouched = append(untouched, fmt.Sprintf("  %s %s", l.Kind, l.Path))
		}
	}
	if len(untouched) > 0 {
		_, _ = fmt.Fprintf(r.stdout, "\nNo structure change on the way affects these, only their recorded version moves:\n%s\n", strings.Join(untouched, "\n"))
	}

	if !dryRun {
		return
	}

	if len(result.Files) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "\nNo files would change, only the recorded versions.\n")
		return
	}

	_, _ = fmt.Fprintf(r.stdout, "\nFiles:\n")
	w := tabwriter.NewWriter(r.stdout, 0, 0, 2, ' ', 0)
	for _, f := range result.Files {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", f.Op, f.Path)
	}
	_ = w.Flush()
}
