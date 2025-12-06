package deploy

import (
	"fmt"
	"io"

	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v5/pkg/middleware/renewtoken"
)

const (
	name             = "deploy"
	shortDescription = "Manage GiantSwarm deployments (apps and config repositories)"
	longDescription  = `Manage GiantSwarm deployment of apps and config repositories to clusters.

This command helps you deploy, undeploy, and check the status of resources
on your cluster. It supports both app deployments and config repository
deployments.`

	examples = `  # Deploy an app with a specific version
  kubectl gs deploy -d my-app@1.0.0-commit

  # Deploy with synchronous reconciliation
  kubectl gs deploy -d --sync my-app@1.0.0-commit

  # Deploy and undeploy on exit (Ctrl+C to undeploy)
  kubectl gs deploy -d -r my-app@1.0.0-commit

  # Deploy interactively (select app and version from catalog)
  kubectl gs deploy -d -i

  # Deploy interactively with a specific app name filter
  kubectl gs deploy -d -i my-app

  # Deploy interactively and choose catalog first (pass empty string to catalog)
  kubectl gs deploy -d -i -c ""

  # Deploy interactively with undeploy-on-exit
  kubectl gs deploy -d -i -r

  # Deploy a config repository
  kubectl gs deploy -t config -d config-repo@branch-ref

  # Deploy a config repository with synchronous reconciliation
  kubectl gs deploy -t config -d --sync config-repo@branch-ref

  # Deploy a config repository with undeploy-on-exit
  kubectl gs deploy -t config -d -r config-repo@branch-ref

  # Undeploy an app
  kubectl gs deploy -u my-app

  # Undeploy a config repository
  kubectl gs deploy -t config -u config-repo

  # Show status of all resources on the cluster
  kubectl gs deploy -s

  # List all apps in the default namespace
  kubectl gs deploy -l apps

  # List all apps in a specific namespace
  kubectl gs deploy -l apps -n my-namespace

  # List all versions for a specific app
  kubectl gs deploy -l versions my-app

  # List all config repositories
  kubectl gs deploy -l configs

  # List all config repositories in a specific namespace
  kubectl gs deploy -t config -l configs -n flux-giantswarm

  # List all available catalogs
  kubectl gs deploy -l catalogs

  # Deploy to a specific namespace
  kubectl gs deploy -d my-app@1.0.0 -n my-namespace

  # Deploy from a specific catalog
  kubectl gs deploy -d my-app@1.0.0 -c my-catalog`
)

type Config struct {
	Logger      micrologger.Logger
	FileSystem  afero.Fs
	ConfigFlags *genericclioptions.RESTClientGetter
	Stderr      io.Writer
	Stdout      io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("%w: %T.Logger must not be empty", ErrInvalidConfig, config)
	}
	if config.FileSystem == nil {
		return nil, fmt.Errorf("%w: %T.FileSystem must not be empty", ErrInvalidConfig, config)
	}
	if config.ConfigFlags == nil {
		return nil, fmt.Errorf("%w: %T.ConfigFlags must not be empty", ErrInvalidConfig, config)
	}
	if config.Stderr == nil {
		return nil, fmt.Errorf("%w: %T.Stderr must not be empty", ErrInvalidConfig, config)
	}
	if config.Stdout == nil {
		return nil, fmt.Errorf("%w: %T.Stdout must not be empty", ErrInvalidConfig, config)
	}

	f := &flag{}

	r := &runner{
		commonConfig: &commonconfig.CommonConfig{
			ConfigFlags: config.ConfigFlags,
		},
		flag:   f,
		logger: config.Logger,
		fs:     config.FileSystem,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name + " [resource@version]",
		Short:   shortDescription,
		Long:    longDescription,
		Example: examples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
