package main

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/cmd"
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

	fs := afero.NewOsFs()

	k8sConfigAccess := clientcmd.NewDefaultPathOptions()

	var rootCommand *cobra.Command
	{
		c := cmd.Config{
			Logger:     logger,
			FileSystem: fs,

			K8sConfigAccess: k8sConfigAccess,
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
