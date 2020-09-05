package apps

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
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

Output columns:

- NAMESPACE: Namespace the App CR is installed in. Corresponds to a Cluster ID.
- NAME: Name of the app being validated.
- VERSION: Version of the app.
- ERRORS: Number of validation errors.	`

	examples = `  # Get an overview of all apps and the number of validation errors per app.
  # Note: This will download the catalog and a spec file (once for each version
  # of an app encountered) and might take a while depending on your network and the
  # number of apps.

  kubectl gs validate apps

  # Narrow down by namespace to validate apps on a specific tenant cluster

    kubectl gs validate apps -n oby63

  # Get a detailed validation report on a specific app on a specific cluster

    kubectl gs validate apps nginx-ingress-controller -n oby63 -o report

  # Get a detailed validation report of a specific app across all tenant clusters
  # the "app" label contains the name of the app in the App Catalog, so we can use --selector for that.

    kubectl gs validate apps --selector=app=nginx-ingress-controller-app -o report

  # Validate the values of an app against a local values schema file. Not using the label
  # selector in this case, because we want a specific instance of an app, so the positional
  # argument can be used to fetch an app by its name.

    kubectl gs validate apps my-nginx-ingress-controller -n oby63 --values-schema-file=values.schema.json`
)

type Config struct {
	Logger micrologger.Logger

	K8sConfigAccess clientcmd.ConfigAccess

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sConfigAccess == nil {
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
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
