package capi

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v3/pkg/commonconfig"
)

const (
	name        = "cluster-api"
	description = "Display information about Cluster API CRDs and controllers"
)

type Config struct {
	Logger      micrologger.Logger
	ConfigFlags *genericclioptions.RESTClientGetter
	Stdout      io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ConfigFlags == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigFlags must not be empty", config)
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
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Aliases: []string{"capi"},
		Short:   description,
		Long:    description,
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
