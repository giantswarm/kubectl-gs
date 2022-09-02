package add

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	app "github.com/giantswarm/kubectl-gs/cmd/gitops/add/app"
	autoup "github.com/giantswarm/kubectl-gs/cmd/gitops/add/automatic-updates"
	enc "github.com/giantswarm/kubectl-gs/cmd/gitops/add/encryption"
	mc "github.com/giantswarm/kubectl-gs/cmd/gitops/add/management-cluster"
	org "github.com/giantswarm/kubectl-gs/cmd/gitops/add/organization"
	wc "github.com/giantswarm/kubectl-gs/cmd/gitops/add/workload-cluster"
)

const (
	name        = "add"
	description = "Add various resources into your GitOps repository"
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

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

	var appCmd *cobra.Command
	{
		c := app.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		appCmd, err = app.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var autoUpdateCmd *cobra.Command
	{
		c := autoup.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		autoUpdateCmd, err = autoup.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var encryptionCmd *cobra.Command
	{
		c := enc.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		encryptionCmd, err = enc.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var mcCmd *cobra.Command
	{
		c := mc.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		mcCmd, err = mc.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var orgCmd *cobra.Command
	{
		c := org.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		orgCmd, err = org.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var wcCmd *cobra.Command
	{
		c := wc.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		wcCmd, err = wc.New(c)
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

	c.AddCommand(appCmd)
	c.AddCommand(autoUpdateCmd)
	c.AddCommand(encryptionCmd)
	c.AddCommand(mcCmd)
	c.AddCommand(orgCmd)
	c.AddCommand(wcCmd)

	return c, nil
}
