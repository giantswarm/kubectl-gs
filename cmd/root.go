package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/cmd/get"
	"github.com/giantswarm/kubectl-gs/cmd/login"
	"github.com/giantswarm/kubectl-gs/cmd/template"
<<<<<<< HEAD
	"github.com/giantswarm/kubectl-gs/pkg/project"
=======
	"github.com/giantswarm/kubectl-gs/cmd/validate"
>>>>>>> 2a223a9... Add validate command
)

const (
	// Hack to set base command name as 'kubectl gs', since
	// cobra splits all the words in the 'usage' field and
	// only prints the first word. The splitting is done by
	// space characters (' '), and we trick it by using a
	// NBSP character (NBSP) between the 2 words.
	name        = "kubectl\u00a0gs"
	description = `Your user-friendly kubectl plug-in for the Giant Swarm control plane.

Get more information at https://github.com/giantswarm/kubectl-gs

Note: We highly recommend setting up 'kgs' as a shorthand.

   alias kgs="kubectl gs"
`
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

	var loginCmd *cobra.Command
	{
		c := login.Config{
			Logger:     config.Logger,
			FileSystem: config.FileSystem,

			K8sConfigAccess: config.K8sConfigAccess,

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

			K8sConfigAccess: config.K8sConfigAccess,

			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		getCmd, err = get.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var validateCmd *cobra.Command
	{
		c := validate.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		validateCmd, err = validate.New(c)
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
		Use:           name,
		Short:         description,
		Long:          description,
		RunE:          r.Run,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       project.Version(),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command %q for %s", args[0], cmd.CommandPath())
			}
			return nil
		},
	}

	f.Init(c)

	c.AddCommand(loginCmd)
	c.AddCommand(templateCmd)
	c.AddCommand(getCmd)
	c.AddCommand(validateCmd)

	return c, nil
}
