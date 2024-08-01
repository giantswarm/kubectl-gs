package update

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v4/cmd/update/app"
	"github.com/giantswarm/kubectl-gs/v4/cmd/update/cluster"
	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
)

const (
	name        = "update"
	description = "Update different types of CRs"
)

type Config struct {
	Logger      micrologger.Logger
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

	var err error

	var appCmd *cobra.Command
	{
		c := app.Config{
			Logger: config.Logger,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		appCmd, err = app.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterCmd *cobra.Command
	{
		c := cluster.Config{
			Logger: config.Logger,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		clusterCmd, err = cluster.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
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
		Aliases: []string{"upgrade"},
		Short:   description,
		Long:    description,
		RunE:    r.Run,
	}

	f.Init(c)

	c.AddCommand(appCmd)
	c.AddCommand(clusterCmd)

	return c, nil
}
