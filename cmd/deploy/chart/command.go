package chart

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware/renewtoken"
)

const (
	commandName = "chart"

	shortDescription = "[DEVELOPMENT] Deploy a Helm chart via Flux OCIRepository and HelmRelease."
	longDescription  = `[DEVELOPMENT] Deploy a Helm chart to a workload cluster via Flux OCIRepository and HelmRelease.

NOTE: This command is currently in development and may change without notice.

Generates OCIRepository and HelmRelease manifests for deploying a Helm chart
from an OCI registry and applies them to the management cluster. The resources
are created in the organization namespace.

Resource names default to <cluster>-<chart-name> and can be overridden with --name.

Use --dry-run to perform server-side validation without persisting resources.
Use --management-cluster to deploy to the MC itself (no --target-cluster needed).`

	examples = `  # Deploy a chart with a specific version
  kubectl gs deploy chart \
      --chart-name hello-world-app \
      --version 1.2.3 \
      --organization acme \
      --target-cluster mycluster01 \
      --target-namespace hello

  # Deploy a chart from a custom registry
  kubectl gs deploy chart \
      --oci-url-prefix oci://example.com/charts/ \
      --chart-name my-chart \
      --version 2.0.0 \
      --organization acme \
      --target-cluster mycluster01 \
      --target-namespace my-namespace

  # Deploy with auto-upgrade on patch versions
  kubectl gs deploy chart \
      --chart-name hello-world-app \
      --version 1.2.3 \
      --auto-upgrade patch \
      --organization acme \
      --target-cluster mycluster01 \
      --target-namespace hello

  # Deploy with custom values
  kubectl gs deploy chart \
      --chart-name hello-world-app \
      --version 1.2.3 \
      --organization acme \
      --target-cluster mycluster01 \
      --target-namespace hello \
      --values-file my-values.yaml

  # Server-side dry-run (validate without applying)
  kubectl gs deploy chart \
      --chart-name hello-world-app \
      --organization acme \
      --target-cluster mycluster01 \
      --target-namespace hello \
      --dry-run

  # Deploy to the management cluster itself
  kubectl gs deploy chart \
      --chart-name hello-world-app \
      --organization acme \
      --target-namespace hello \
      --management-cluster`
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
		fileSystem: config.FileSystem,
		flag:       f,
		logger:     config.Logger,
		stderr:     config.Stderr,
		stdout:     config.Stdout,
	}

	c := &cobra.Command{
		Use:     commandName,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Args:    cobra.ExactArgs(0),
		RunE:    r.Run,
		Hidden:  true,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
