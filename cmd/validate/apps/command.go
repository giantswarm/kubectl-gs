package apps

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware/renewtoken"
)

var (
	aliases = []string{"app"}
)

const (
	name             = "apps <app-name>"
	shortDescription = "Validate apps (App CRs)"
	longDescription  = `Validate apps (App CRs)

Validates apps, ensuring they are configured correctly and showing if there are
any validation errors.

Without any flags it provides a quick overview of all apps installed across
all clusters. Use flags to narrow down which apps are being validated, or to
get more detailed information about any validation errors.

This command requires a Helm 3 binary called 'helm' in your $PATH.

Output columns:

- NAMESPACE: Namespace the App CR is installed in. This will generally be the cluster or organization namespace.
- NAME: Name of the app being validated.
- VERSION: Version of the app.
- ERRORS: Number of validation errors.	`

	examples = `  # Get an overview of all apps and the number of validation errors per app.
  # Note: This will download the catalog and a spec file (once for each version
  # of an app encountered) and might take a while depending on your network and the
  # number of apps.

  kubectl gs validate apps

  # Narrow down by namespace to validate apps on a specific workload cluster

    kubectl gs validate apps \
      --namespace oby63

  # Get a detailed validation report on a specific app on a specific cluster

    kubectl gs validate app \
      --namespace oby63 \
      ingress-nginx \
      --output report

  # Get a detailed validation report of a specific app across all workload clusters
  # the "app" label contains the name of the app in the Catalog, so we can use --selector for that.

    kubectl gs validate apps \
      --selector app.kubernetes.io/name=ingress-nginx \
      --output report

  # Validate the values of an app against a local values schema file. Not using the label
  # selector in this case, because we want a specific instance of an app, so the positional
  # argument can be used to fetch an app by its name.

    kubectl gs validate app
      --namespace oby63 \
      ingress-nginx \
      --values-schema-file=values.schema.json`
)

// Config are the configuration that New takes to create an instance of this command.
type Config struct {
	Logger      micrologger.Logger
	ConfigFlags *genericclioptions.RESTClientGetter

	Stderr io.Writer
	Stdout io.Writer
}

// New takes a Config and returns an instance of this command.
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
		Aliases: aliases,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
