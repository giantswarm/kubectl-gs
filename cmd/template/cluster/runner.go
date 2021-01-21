package cluster

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/id"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	clusterCRFileName = "clusterCR"
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
	var err error

	var config provider.ClusterCRsConfig
	{
		config = provider.ClusterCRsConfig{
			FileName:       clusterCRFileName,
			ClusterID:      r.flag.ClusterID,
			ExternalSNAT:   r.flag.ExternalSNAT,
			MasterAZ:       r.flag.MasterAZ,
			Description:    r.flag.Name,
			Owner:          r.flag.Owner,
			PodsCIDR:       r.flag.PodsCIDR,
			ReleaseVersion: r.flag.Release,
			Namespace:      metav1.NamespaceDefault,
		}

		if config.ClusterID == "" {
			config.ClusterID = id.Generate()
		}

		// Remove leading 'v' from release flag input.
		config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

		config.Labels, err = clusterlabels.Parse(r.flag.Label)
		if err != nil {
			return microerror.Mask(err)
		}

		if r.flag.Provider == key.ProviderAzure {
			config.Namespace = key.OrganizationNamespaceFromName(config.Owner)
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

	switch r.flag.Provider {
	case key.ProviderAWS:
		err = provider.WriteAWSTemplate(output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAzure:
		err = provider.WriteAzureTemplate(output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
