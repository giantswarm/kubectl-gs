package check

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"text/tabwriter"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

type runner struct {
	flag   *flag
	fs     *afero.Afero
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
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

	md, err := metadata.Load(r.fs, repoPath)
	if metadata.IsNotFound(err) {
		if !r.flag.Adopt {
			return microerror.Maskf(
				metadataNotFoundError,
				"%s holds no %s, so there is nothing to check yet.\nRun `kubectl gs gitops check --adopt` to record what the repository already contains.",
				repoPath,
				metadata.FileName,
			)
		}
		md = nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if r.flag.Adopt {
		return r.adopt(repoPath, md, dryRun)
	}

	return r.report(md)
}

// adopt records the layers of a repository that is not tracking its structure
// version yet. Layers already recorded keep the version they were recorded
// with, so re-running adoption never bumps them.
func (r *runner) adopt(repoPath string, md *metadata.RepositoryMetadata, dryRun bool) error {
	result, err := metadata.Adopt(r.fs, repoPath)
	if err != nil {
		return microerror.Mask(err)
	}

	fresh := md == nil
	if fresh {
		md = metadata.New()
	}

	added := md.MergeMissing(result.Metadata.Layers)

	if len(added) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "Nothing to adopt: all %d detected layer(s) are already recorded.\n", len(result.Metadata.Layers))
		return nil
	}

	_, _ = fmt.Fprintf(r.stdout, "Detected %d layer(s) to record at structure version %d:\n\n", len(added), metadata.StructureVersion)
	printKindCounts(r.stdout, added)

	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintf(r.stdout, "\nSkipped %d directories that do not look like a generated layer:\n", len(result.Skipped))
		for _, s := range result.Skipped {
			_, _ = fmt.Fprintf(r.stdout, "  %s\n", s)
		}
	}

	body, err := metadata.Render(md)
	if err != nil {
		return microerror.Mask(err)
	}

	if dryRun {
		_, _ = fmt.Fprintf(r.stdout, "\n%s\n%s\n", metadata.FilePath(repoPath), string(body))
		return nil
	}

	err = metadata.Save(r.fs, repoPath, md)
	if err != nil {
		return microerror.Mask(err)
	}

	_, _ = fmt.Fprintf(r.stdout, "\nWrote %s\n", metadata.FilePath(repoPath))
	_, _ = fmt.Fprintf(r.stdout, "\nNote: adoption assumes these layers are current. Drift from before this\npoint is not detectable.\n")

	return nil
}

// report prints how each recorded layer compares to the structure version this
// kubectl-gs generates, and fails when anything is behind.
func (r *runner) report(md *metadata.RepositoryMetadata) error {
	repoOutdated := md.StructureVersion < metadata.StructureVersion

	if repoOutdated {
		_, _ = fmt.Fprintf(r.stdout, "repository structure version: %d (current: %d)\n", md.StructureVersion, metadata.StructureVersion)
	} else {
		_, _ = fmt.Fprintf(r.stdout, "repository structure version: %d (up to date)\n", md.StructureVersion)
	}

	if len(md.Layers) == 0 {
		_, _ = fmt.Fprintf(r.stdout, "\nNo layers recorded yet.\n")
	} else {
		md.Sort()

		_, _ = fmt.Fprintln(r.stdout)
		w := tabwriter.NewWriter(r.stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "LAYER\tPATH\tVERSION\tSTATUS")
		for _, l := range md.Layers {
			status := "up to date"
			if l.StructureVersion < metadata.StructureVersion {
				status = fmt.Sprintf("outdated (current %d)", metadata.StructureVersion)
			} else if l.StructureVersion > metadata.StructureVersion {
				// The repository was touched by a newer kubectl-gs than the
				// one running now. Worth saying so rather than calling it
				// up to date.
				status = "newer than this kubectl-gs"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", l.Kind, l.Path, l.StructureVersion, status)
		}
		_ = w.Flush()
	}

	outdated := md.OutdatedLayers()

	_, _ = fmt.Fprintln(r.stdout)
	switch {
	case len(outdated) > 0:
		_, _ = fmt.Fprintf(r.stdout, "%d layer(s) out of sync.\n", len(outdated))
	case repoOutdated:
		_, _ = fmt.Fprintf(r.stdout, "The repository itself is out of sync.\n")
	case len(md.Layers) == 0:
		_, _ = fmt.Fprintf(r.stdout, "The repository is up to date. Run with --adopt to record the layers it already holds.\n")
	default:
		_, _ = fmt.Fprintf(r.stdout, "All %d layer(s) are up to date.\n", len(md.Layers))
	}

	if len(outdated) > 0 || repoOutdated {
		return microerror.Maskf(
			outOfSyncError,
			"the repository was generated with an older repository structure than this kubectl-gs produces (version %d).\nSee https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md for what changed.",
			metadata.StructureVersion,
		)
	}

	return nil
}

// printKindCounts prints how many layers of each kind were found.
func printKindCounts(out io.Writer, layers []metadata.Layer) {
	counts := map[string]int{}
	for _, l := range layers {
		counts[l.Kind]++
	}

	kinds := make([]string, 0, len(counts))
	for k := range counts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, k := range kinds {
		_, _ = fmt.Fprintf(w, "  %s\t%d\n", k, counts[k])
	}
	_ = w.Flush()
}
