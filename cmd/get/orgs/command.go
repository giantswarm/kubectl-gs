// Package org defines the 'kubectl gs get orgs' command.
package orgs

import (
	"io"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"

	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name = "orgs"

	shortDescription = "Display one or many organizations"
	longDescription  = `Display one or many organizations.

Output columns:

- NAME: The organization's name, or: name of the Organization (organizations.security.giantswarm.io) resource.
- NAMESPACE: The namespace belonging to this organization.
`

	examples = `  # List all organizations you have access to
  kgs get orgs

  # Get one specific organization
  kgs get organization acme

  Note: 'kgs' is an alias for 'kubectl gs'.`
)

var (
	aliases = []string{"organizations", "org"}
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	ConfigFlags *genericclioptions.RESTClientGetter

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
	if config.ConfigFlags == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sConfigAccess must not be empty", config)
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
		logger: config.Logger,
		fs:     config.FileSystem,

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Aliases: aliases,
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
