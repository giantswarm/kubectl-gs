package app

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	name = "app"

	shortDescription = "Adds a new Application to your GitOps directory structure"
	longDescription  = `Adds a new Application to your GitOps directory structure.

app \
--app <chart_name> \
[--base <path_to_base>] \
[--name <app_name>] \
[--oci-url-prefix <oci_repository>] \
--target-namespace <app_namespace> \
--management-cluster <mc_code_name> \
--organization <org_name> \
[--user-configmap <path_to_values_yaml>] \
[--user-secret <path_to_values_yaml>] \
--version <chart_version> \
--workload-cluster <wc_name>

The app is deployed with a Flux OCIRepository, pulling the chart from
--oci-url-prefix, and a HelmRelease installing it in the workload cluster.
With --in-cluster, the HelmRelease installs it in the management cluster
instead. Values from --user-configmap and --user-secret are added to the
HelmRelease's valuesFrom.

With --base, the app comes from the base, whose HelmRelease must be named
${cluster_name}-<app_name> and list valuesFrom, as the user values are
appended to it.

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.`

	examples = `  # Add hello-world App to dummy Workload Cluster
  kubectl gs gitops add app \
  --app hello-world \
  --name hello-world \
  --target-namespace default \
  --management-cluster demomc \
  --organization demoorg \
  --version 0.3.0 \
  --workload-cluster demowc

  # Add hello-world App to dummy Workload Cluster, include user configuration
  kubectl gs gitops add app \
  --app hello-world \
  --name hello-world \
  --target-namespace default \
  --management-cluster demomc \
  --organization demoorg \
  --user-configmap /tmp/hello-world-values.yaml \
  --version 0.3.0 \
  --workload-cluster demowc

  # Add hello-world App from the base
  kubectl gs gitops add app \
  --base base/apps/hello-world \
  --name hello-world \
  --management-cluster demomc \
  --organization demoorg \
  --user-configmap /tmp/hello-world-values.yaml \
  --workload-cluster demowc`
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
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Deprecated: "currently without replacement",
		Use:        name,
		Short:      shortDescription,
		Long:       longDescription,
		Example:    examples,
		RunE:       r.Run,
	}

	f.Init(c)

	return c, nil
}
