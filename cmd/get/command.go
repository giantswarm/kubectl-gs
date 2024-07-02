package get

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v4/cmd/get/apps"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/capi"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/catalogs"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/clusters"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/nodepools"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/orgs"
	"github.com/giantswarm/kubectl-gs/v4/cmd/get/releases"
	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
)

const (
	name        = "get"
	description = "Get various types of resources."
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
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
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

	var appsCmd *cobra.Command
	{
		c := apps.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		appsCmd, err = apps.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var catalogsCmd *cobra.Command
	{
		c := catalogs.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		catalogsCmd, err = catalogs.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterApiCmd *cobra.Command
	{
		c := capi.Config{
			Logger:      config.Logger,
			ConfigFlags: config.ConfigFlags,
			Stdout:      config.Stdout,
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

			ConfigFlags: config.ConfigFlags,

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

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		nodepoolsCmd, err = nodepools.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releasesCmd *cobra.Command
	{
		c := releases.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,
			Stderr:      config.Stderr,
			Stdout:      config.Stdout,
		}

		releasesCmd, err = releases.New(c)

		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var orgsCmd *cobra.Command
	{
		c := orgs.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,
			Stderr:      config.Stderr,
			Stdout:      config.Stdout,
		}

		orgsCmd, err = orgs.New(c)

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
		RunE:  r.Run,
	}

	f.Init(c)

	c.AddCommand(appsCmd)
	c.AddCommand(catalogsCmd)
	c.AddCommand(clusterApiCmd)
	c.AddCommand(clustersCmd)
	c.AddCommand(nodepoolsCmd)
	c.AddCommand(releasesCmd)
	c.AddCommand(orgsCmd)

	return c, nil
}
