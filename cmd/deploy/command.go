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
	name  = "deploy"
	shortDescription = "Manage GiantSwarm deployments (apps and config repositories)"
	longDescription  = `Manage GiantSwarm deployment of apps and config repositories to clusters.

This command helps you deploy, undeploy, and check the status of resources
on your cluster. It supports both app deployments and config repository
deployments.`

	examples = `  # Deploy an app with a specific version
  kubectl gs deploy -d my-app@1.0.0-commit

  # Deploy a config repository
  kubectl gs deploy -t config -d config-repo@branch-ref

  # Undeploy an app
  kubectl gs deploy -u my-app

  # Undeploy a config repository
  kubectl gs deploy -t config -u config-repo

  # Show status of all resources on the cluster
  kubectl gs deploy -s

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
