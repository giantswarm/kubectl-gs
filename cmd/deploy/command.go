package deploy

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v5/cmd/deploy/chart"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
)

const (
	name        = "deploy"
	description = "Deploy resources to a Giant Swarm management cluster"
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

	var err error

	var chartCmd *cobra.Command
	{
		c := chart.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		chartCmd, err = chart.New(c)
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
		Use:    name,
		Short:  description,
		Long:   description,
		RunE:   r.Run,
		Hidden: true,
	}

	f.Init(c)

	c.AddCommand(chartCmd)

	return c, nil
}
