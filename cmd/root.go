package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v2/cmd/get"
	"github.com/giantswarm/kubectl-gs/v2/cmd/gitops"
	"github.com/giantswarm/kubectl-gs/v2/cmd/install"
	"github.com/giantswarm/kubectl-gs/v2/cmd/login"
	"github.com/giantswarm/kubectl-gs/v2/cmd/selfupdate"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template"
	"github.com/giantswarm/kubectl-gs/v2/cmd/update"
	"github.com/giantswarm/kubectl-gs/v2/cmd/validate"
	"github.com/giantswarm/kubectl-gs/v2/pkg/project"
)

const (
	name        = "kubectl-gs"
	description = `Your user-friendly kubectl plug-in for the Giant Swarm management cluster.

Get more information at https://docs.giantswarm.io/use-the-api/kubectl-gs/
`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	Stderr io.Writer
	Stdout io.Writer
	Stdin  *os.File
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
	if config.Stdin == nil {
		config.Stdin = os.Stdin
	}

	var err error

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:                name,
		Short:              description,
		Long:               description,
		RunE:               r.Run,
		PersistentPostRunE: r.PersistentPostRun,
		SilenceUsage:       true,
		SilenceErrors:      true,
		Version:            project.Version(),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command %q for %s", args[0], cmd.CommandPath())
			}
			return nil
		},
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "kubectl gs",
		},
	}
	f.Init(c)

	var loginCmd *cobra.Command
	{
		c := login.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: &f.config,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		loginCmd, err = login.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var templateCmd *cobra.Command
	{
		c := template.Config{
			Logger: config.Logger,

			ConfigFlags: &f.config,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		templateCmd, err = template.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var getCmd *cobra.Command
	{
		c := get.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: &f.config,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		getCmd, err = get.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var gitopsCmd *cobra.Command
	{
		c := gitops.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			ConfigFlags: &f.config,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		gitopsCmd, err = gitops.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var installCmd *cobra.Command
	{
		c := install.Config{
			Logger:      config.Logger,
			ConfigFlags: &f.config,
			Stderr:      config.Stderr,
			Stdout:      config.Stdout,
		}

		installCmd, err = install.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var validateCmd *cobra.Command
	{
		c := validate.Config{
			Logger:      config.Logger,
			ConfigFlags: &f.config,
			Stderr:      config.Stderr,
			Stdout:      config.Stdout,
		}

		validateCmd, err = validate.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var updateCmd *cobra.Command
	{
		c := update.Config{
			Logger:      config.Logger,
			ConfigFlags: &f.config,
			Stderr:      config.Stderr,
			Stdout:      config.Stdout,
		}

		updateCmd, err = update.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var selfUpdateCmd *cobra.Command
	{
		c := selfupdate.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,
			Stderr:     config.Stderr,
			Stdout:     config.Stdout,
		}

		selfUpdateCmd, err = selfupdate.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	c.AddCommand(getCmd)
	c.AddCommand(gitopsCmd)
	c.AddCommand(installCmd)
	c.AddCommand(loginCmd)
	c.AddCommand(templateCmd)
	c.AddCommand(updateCmd)
	c.AddCommand(validateCmd)
	c.AddCommand(selfUpdateCmd)

	return c, nil
}
