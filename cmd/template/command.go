package template

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/app"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/catalog"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/networkpool"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/nodepool"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/organization"
	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
)

const (
	name        = "template"
	description = "Template different types of CRs"
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
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		appCmd, err = app.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appcatalogCmd *cobra.Command
	{
		c := catalog.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		appcatalogCmd, err = catalog.New(c)
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

	var nodepoolCmd *cobra.Command
	{
		c := nodepool.Config{
			Logger: config.Logger,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		nodepoolCmd, err = nodepool.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkpoolCmd *cobra.Command
	{
		c := networkpool.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		networkpoolCmd, err = networkpool.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var organizationCmd *cobra.Command
	{
		c := organization.Config{
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		organizationCmd, err = organization.New(c)
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
		Use:   name,
		Short: description,
		Long:  description,
		Args:  cobra.NoArgs,
		RunE:  r.Run,
	}

	f.Init(c)

	c.AddCommand(appCmd)
	c.AddCommand(appcatalogCmd)
	c.AddCommand(clusterCmd)
	c.AddCommand(networkpoolCmd)
	c.AddCommand(nodepoolCmd)
	c.AddCommand(organizationCmd)

	return c, nil
}
