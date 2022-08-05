package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/cmd"
	"github.com/giantswarm/kubectl-gs/pkg/errorprinter"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	// Disable color on Windows, as it doesn't work in CMD.exe.
	if runtime.GOOS == "windows" {
		color.NoColor = true
	}

	err := mainE(context.Background())
	if err != nil {
		ep := errorprinter.New(errorprinter.Config{
			StackTrace: isDebugMode(),
		})
		fmt.Fprintln(os.Stderr, ep.Format(err))
		os.Exit(1)
	}
}

func mainE(ctx context.Context) error {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}
		logger, err = micrologger.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var activationLogger micrologger.Logger
	{
		c := micrologger.ActivationLoggerConfig{
			Underlying: logger,

			Activations: map[string]interface{}{
				micrologger.KeyLevel:     "error",
				micrologger.KeyVerbosity: 1,
			},
		}

		activationLogger, err = micrologger.NewActivation(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	fs := afero.NewOsFs()
	var rootCommand *cobra.Command
	{
		c := cmd.Config{
			Logger:     activationLogger,
			FileSystem: fs,
		}

		rootCommand, err = cmd.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = rootCommand.Execute()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func isDebugMode() bool {
	for _, arg := range os.Args {
		if arg == "--debug" {
			return true
		}
	}

	return false
}
