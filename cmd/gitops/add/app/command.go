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
	name  = "app"
	alias = "app"

	shortDescription = "Adds a new Application to your GitOps directory structure"
	longDescription  = `Adds a new Application to your GitOps directory structure.

app \
--app <app_to_install> \
[--base <path_to_base>] \
--catalog <app_catalog> \
[--name <app_cr_name>] \
--namespace <app_namespace> \
--management-cluster <mc_code_name> \
--organization <org_name> \
[--user-configmap <path_to_values_yaml>] \
[--user-secret <path_to_values_yaml>] \
--version <app_version> \
--workload-cluster <wc_name>

It respects the Giantswarm's GitOps repository structure recommendation:
https://github.com/giantswarm/gitops-template/blob/main/docs/repo_structure.md.

Steps it implements:
https://github.com/giantswarm/gitops-template/blob/main/docs/apps/add_appcr.md`

	examples = `  # Add hello-world App to dummy Workload Cluster
  kubectl gs gitops add app \
  --app hello-world-app \
  --catalog giantswarm \
  --name hello-world \
  --namespace default \
  --management-cluster demowc \
  --organization demoorg \
  --version 0.3.0 \
  --workload-cluster demowc

  # Add hello-world App to dummy Workload Cluster, include user configuration
  kubectl gs gitops add app \
  --app hello-world-app \
  --catalog giantswarm \
  --name hello-world \
  --namespace default \
  --management-cluster demomc \
  --organization demoorg \
  --user-configmap /tmp/hello-world-app-values.yaml \
  --version 0.3.0 \
  --workload-cluster demowc

  # Add hello-world App from the base
  kubectl gs gitops add app \
  --base base/apps/hello-world \
  --management-cluster demomc \
  --organization demoorg \
  --user-configmap /tmp/hello-world-app-values.yaml \
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
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Aliases: []string{alias},
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
