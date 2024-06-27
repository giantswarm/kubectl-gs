package app

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v3/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v3/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v3/pkg/middleware/renewtoken"
)

const (
	name = "app --name <app-name> --namespace <cluster-namespace> --version <updated-app-version>"

	shortDescription = "Update App CR."
	longDescription  = `Update App CR.

Updates given app with the provided values.

Options:
  --name <name>              App CR name to update.
  --namespace <cluster>      Cluster to update the app on.
  --version <version>        New version to update the app to.`

	examples = `  # Display this help
kubectl gs update app --help

# Update app version
kubectl gs update app --name hello-world-app --namespace ab01c --version 0.2.0`
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
	if config.ConfigFlags == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigFlags must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		commonConfig: &commonconfig.CommonConfig{
			ConfigFlags: config.ConfigFlags,
		},
		flag:   f,
		logger: config.Logger,

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Args:    cobra.MatchAll(cobra.ExactArgs(0), cobra.OnlyValidArgs),
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
