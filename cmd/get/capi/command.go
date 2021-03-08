package capi

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	name        = "cluster-api"
	description = "Display information about Cluster API CRDs and controllers"
)

type Config struct {
	Logger          micrologger.Logger
	K8sConfigAccess clientcmd.ConfigAccess
	Stdout          io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sConfigAccess == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sConfigAccess must not be empty", config)
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
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
