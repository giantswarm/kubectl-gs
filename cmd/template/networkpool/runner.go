package networkpool

import (
	"context"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v3/cmd/template/networkpool/provider"
	"github.com/giantswarm/kubectl-gs/v3/internal/key"
)

const (
	networkPoolCRFileName = "networkpoolCR"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	var config provider.NetworkPoolCRsConfig
	{
		config = provider.NetworkPoolCRsConfig{
			CIDRBlock:       r.flag.CIDRBlock,
			FileName:        networkPoolCRFileName,
			NetworkPoolName: r.flag.NetworkPoolName,
			Organization:    r.flag.Organization,
		}

		if config.NetworkPoolName == "" {
			generatedName, err := key.GenerateName()
			if err != nil {
				return microerror.Mask(err)
			}

			config.NetworkPoolName = generatedName
		}
	}

	var output *os.File
	{
		if r.flag.Output == "" {
			output = os.Stdout
		} else {
			f, err := os.Create(r.flag.Output)
			if err != nil {
				return microerror.Mask(err)
			}
			defer f.Close()

			output = f
		}
	}

	err = provider.WriteTemplate(output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
