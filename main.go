package main

import (
	"context"
	"fmt"

	"github.com/giantswarm/kubectl-gs/cmd"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

func main() {
	err := mainE(context.Background())
	if err != nil {
		panic(fmt.Sprintf("%#v\n", microerror.JSON(err)))
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

	var rootCommand *cobra.Command
	{
		c := cmd.Config{
			Logger: logger,
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
