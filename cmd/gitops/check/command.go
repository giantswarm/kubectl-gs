package check

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name = "check"

	shortDescription = "Check whether the GitOps repository is up to date with the repository structure."
	longDescription  = `Check whether the GitOps repository is up to date with the repository structure.

Every time ` + "`kubectl gs gitops`" + ` generates something, it records what it created
and the version of the repository structure it used in a ` + "`.gitops-metadata.yaml`" + `
file at the repository root. This command compares those records against the
structure version this kubectl-gs produces, and reports the parts of the
repository that were generated with an older one.

It exits non-zero when anything is out of date, so it can be used as a check in
the repository's own CI.

Repositories created before kubectl-gs started recording this hold no metadata
file. Run the command with --adopt once to record what is already there. Note
that adoption assumes the existing layers are current, so it cannot recover
drift that happened before it ran.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.`

	examples = `  # Check the repository at the current directory
  kubectl gs gitops check

  # Check the repository at a given location
  kubectl gs gitops check --local-path /tmp/gitops-demo

  # Record what a pre-existing repository already holds
  kubectl gs gitops check --local-path /tmp/gitops-demo --adopt

  # See what adoption would record, without writing it
  kubectl gs gitops check --local-path /tmp/gitops-demo --adopt --dry-run`
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
		flag:   f,
		fs:     &afero.Afero{Fs: config.FileSystem},
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
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
