package login

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
)

const (
	name             = "login [K8s API URL | Web UI URL | Existing GS Context Name]"
	shortDescription = "Logs into an installation's Kubernetes API"
	longDescription  = `Log into an installation's Kubernetes API.

You can use as an argument:
  * Your installation's Kubernetes API URL, e. g. 'https://g8s.test.eu-west-1.aws.gigantic.io'
  * Your Web UI URL, e. g. 'https://happa.g8s.test.eu-west-1.aws.gigantic.io'
  * An existing Giant Swarm specific kubectl context name, e. g. 'gs-test'
`
	examples = `  # See on which installation you're logged in currently.
  kgs login

  # Log in using your K8s API URL.
  kgs login https://g8s.test.eu-west-1.aws.gigantic.io

  # Log in using your Web UI URL.
  kgs login https://happa.g8s.test.eu-west-1.aws.gigantic.io

  # Log in using a GS specific context name.
  kgs login gs-test

  # Or even shorter
  kgs login test

  Note: 'kgs' is an alias for 'kubectl gs'.
`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	K8sConfigAccess clientcmd.ConfigAccess

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
	if config.K8sConfigAccess == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigAccess must not be empty", config)
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

		k8sConfigAccess: config.K8sConfigAccess,

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
