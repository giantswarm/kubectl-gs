package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/telemetrydeck-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v5/cmd/get"
	"github.com/giantswarm/kubectl-gs/v5/cmd/gitops"
	"github.com/giantswarm/kubectl-gs/v5/cmd/login"
	"github.com/giantswarm/kubectl-gs/v5/cmd/selfupdate"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template"
	"github.com/giantswarm/kubectl-gs/v5/cmd/update"
	"github.com/giantswarm/kubectl-gs/v5/cmd/validate"
	"github.com/giantswarm/kubectl-gs/v5/pkg/project"
)

const (
	name        = "kubectl-gs"
	description = `Your user-friendly kubectl plug-in for the Giant Swarm management cluster.

Get more information at https://docs.giantswarm.io/use-the-api/kubectl-gs/
`
	telemetrydeckAppID = "4539763B-A291-4835-B832-9BEB80CA7039"

	telemetryOptOutVariable = "KUBECTL_GS_TELEMETRY_OPTOUT"
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
		Use:   name,
		Short: description,
		Long:  description,

		// Called for every subcommand execution
		// to track command usage.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if os.Getenv(telemetryOptOutVariable) != "" {
				return
			}

			logger := log.New(os.Stdout, "", 0) // to bring telemetry errors to the surface
			tdClient, err := telemetrydeck.NewClient(telemetrydeckAppID,
				telemetrydeck.WithLogger(logger),
			)
			if err != nil {
				log.Printf("error creating telemetrydeck client: %s", err)
			} else {
				err = tdClient.SendSignal(context.Background(), "GiantSwarm.command", map[string]interface{}{
					"appVersion": project.Version(),
					"command":    cmd.CommandPath(),
				})
				if err != nil {
					log.Printf("error sending telemetrydeck signal: %s", err)
				}
			}

			// TODO: remove after testing
			fmt.Printf("User ID: %q\n", tdClient.UserID())
			fmt.Printf("User ID hash: %q\n\n", tdClient.UserIDHash())
		},

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
	c.AddCommand(loginCmd)
	c.AddCommand(templateCmd)
	c.AddCommand(updateCmd)
	c.AddCommand(validateCmd)
	c.AddCommand(selfUpdateCmd)

	return c, nil
}
