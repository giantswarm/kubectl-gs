package networkpool

import (
	"context"
	"io"
	"os"

	"github.com/giantswarm/apiextensions/v2/pkg/id"

	"github.com/giantswarm/kubectl-gs/cmd/template/networkpool/provider"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
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
			Owner:           r.flag.Owner,
		}

		if config.NetworkPoolName == "" {
			config.NetworkPoolName = id.Generate()
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
