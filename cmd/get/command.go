package get

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/cmd/get/capi"
	"github.com/giantswarm/kubectl-gs/cmd/get/clusters"
	"github.com/giantswarm/kubectl-gs/cmd/get/nodepools"
)

const (
	name        = "get"
	description = "Get various types of resources."
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	K8sConfigAccess clientcmd.ConfigAccess

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
	if config.K8sConfigAccess == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sConfigAccess must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	var err error

	var clusterApiCmd *cobra.Command
	{
		c := capi.Config{
			Logger:          config.Logger,
			K8sConfigAccess: config.K8sConfigAccess,
			Stdout:          config.Stdout,
		}

		clusterApiCmd, err = capi.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clustersCmd *cobra.Command
	{
		c := clusters.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			K8sConfigAccess: config.K8sConfigAccess,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		clustersCmd, err = clusters.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var nodepoolsCmd *cobra.Command
	{
		c := nodepools.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			K8sConfigAccess: config.K8sConfigAccess,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		nodepoolsCmd, err = nodepools.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:   name,
		Short: description,
		Long:  description,
		RunE:  r.Run,
	}

	f.Init(c)

	c.AddCommand(clusterApiCmd)
	c.AddCommand(clustersCmd)
	c.AddCommand(nodepoolsCmd)

	return c, nil
}
