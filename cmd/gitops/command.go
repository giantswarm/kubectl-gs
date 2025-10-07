package gitops

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v5/cmd/gitops/add"
	"github.com/giantswarm/kubectl-gs/v5/cmd/gitops/initialize"
)

const (
	name        = "gitops"
	description = "GitOps swissknife"
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
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	var err error

	var addCmd *cobra.Command
	{
		c := add.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: config.ConfigFlags,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		addCmd, err = add.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var initCmd *cobra.Command
	{
		c := initialize.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		initCmd, err = initialize.New(c)
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

	c.AddCommand(addCmd)
	c.AddCommand(initCmd)

	return c, nil
}
