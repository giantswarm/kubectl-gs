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

Use --dry-run to see which files would change without writing anything. It runs
the migrations against an in-memory copy of the repository. Run
` + "`kubectl gs gitops check`" + ` first to see which parts are behind, and
` + "`kubectl gs gitops check --adopt`" + ` for a repository that holds no metadata
file yet.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.`

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
		dirtyPaths: dirtyPaths,
		flag:       f,
		fs:         &afero.Afero{Fs: config.FileSystem},
		logger:     config.Logger,
		migrations: migration.Migrations,
		stderr:     config.Stderr,
		stdout:     config.Stdout,
		target:     metadata.StructureVersion,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
