package upgrade

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/migration"
)

const (
	name = "upgrade"

	shortDescription = "Upgrade the GitOps repository to the repository structure this kubectl-gs produces."
	longDescription  = `Upgrade the GitOps repository to the repository structure this kubectl-gs produces.

Every time ` + "`kubectl gs gitops`" + ` generates something, it records the version of
the repository structure it used in a ` + "`.gitops-metadata.yaml`" + ` file at the
repository root. This command takes each part of the repository recorded with an
older version and rewrites it, one structure version at a time, to the version
this kubectl-gs produces. It then records the new version in the metadata file.

The changes are made to the files as they are, so changes made to them by hand
are kept wherever the structure change allows it. There is no rollback of its
own: a failed upgrade is undone with git. The command therefore refuses to run
migrations on a repository with uncommitted changes, or one git does not track,
so that undoing one never loses work. Pass --force to upgrade anyway. An upgrade
that only moves the recorded versions, touching no file but the metadata file,
is not held up by uncommitted changes. Review the result, e.g. with ` + "`git diff`" + `,
before committing it.

Some structure changes can't be made safely in files alone, as Flux has to
apply them in a given order, over more than one commit. The upgrade makes the
part that is safe to push at once, and prints the steps left to do by hand.
The upgrade to version 3 moves the workload clusters' mapi directories, adds
the organisations' namespace Kustomizations, and protects the management
cluster's keys with a kustomization.yaml, but leaves moving the clusters'
Kustomizations into the organisations' namespaces to you. It refuses a
cluster whose Kustomization prunes what out-of-band/ holds, as Flux would
delete it from the management cluster, and a cluster with a kustomization
file of its own next to mapi/, which doesn't list what moves up. It also
refuses a cluster whose Kustomization applies mapi/ with prune: true until
every object there has the kustomize.toolkit.fluxcd.io/prune: disabled
annotation, as objects that change path can be deleted and created again
while they move: add it, let Flux apply it, then upgrade in the next commit.
It refuses a cluster, too, while a .sops.yaml rule picks the key for files
under mapi/ that it doesn't pick once they have moved up: change the rule to
match both places first. It never changes .sops.yaml.
See
https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/upgrade-fork.md#check-and-upgrade-the-structure-version

Files git ignores, such as decrypted secrets, are never changed, as git
cannot restore them: they are left where they are, so move them yourself. The
upgrade refuses when one has to change, until you have changed it by hand.

Use --dry-run to see which files would change without writing anything. It runs
the migrations against an in-memory copy of the repository. Run
` + "`kubectl gs gitops check`" + ` first to see which parts are behind, and
` + "`kubectl gs gitops check --adopt`" + ` for a repository that holds no metadata
file yet.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md#structure-version.`

	examples = `  # Upgrade the repository at the current directory
  kubectl gs gitops upgrade

  # See what upgrading the repository at a given location would change
  kubectl gs gitops upgrade --local-path /tmp/gitops-demo --dry-run

  # Upgrade a repository with uncommitted changes, accepting that a failed
  # upgrade cannot then be undone with git alone
  kubectl gs gitops upgrade --local-path /tmp/gitops-demo --force`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		dirtyPaths:   dirtyPaths,
		ignoredPaths: ignoredPaths,
		flag:         f,
		fs:           &afero.Afero{Fs: config.FileSystem},
		logger:       config.Logger,
		migrations:   migration.Migrations,
		stderr:       config.Stderr,
		stdout:       config.Stdout,
		target:       metadata.StructureVersion,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		// The repository is given with --local-path. A positional argument
		// would otherwise be ignored, and the current directory upgraded
		// instead of the repository the user meant.
		Args: cobra.NoArgs,
		RunE: r.Run,
	}

	f.Init(c)

	return c, nil
}
