package cluster

import (
	"context"
	"io"
	"os"
	"sort"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	clusterCRName = "clusterCR"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.MasterAZ, func(i, j int) bool {
		return r.flag.MasterAZ[i] < r.flag.MasterAZ[j]
	})

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

	var err error
	switch r.flag.Provider {
	case key.ProviderAWS:
		err = writeAWSTemplate(output, clusterCRName, r.flag)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAzure:
		err = writeAzureTemplate(output, clusterCRName, r.flag)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
