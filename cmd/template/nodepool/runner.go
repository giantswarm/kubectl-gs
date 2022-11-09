package nodepool

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/nodepool/provider"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
)

const (
	nodePoolCRFileName = "nodepoolCR"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer
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

	var config provider.NodePoolCRsConfig
	{
		config = provider.NodePoolCRsConfig{
			AWSInstanceType:                     r.flag.AWSInstanceType,
			FileName:                            nodePoolCRFileName,
			ClusterName:                         r.flag.ClusterName,
			Description:                         r.flag.Description,
			VMSize:                              r.flag.AzureVMSize,
			AzureUseSpotVms:                     r.flag.AzureUseSpotVms,
			AzureSpotMaxPrice:                   r.flag.AzureSpotVMsMaxPrice,
			MachineDeploymentSubnet:             r.flag.MachineDeploymentSubnet,
			NodesMax:                            r.flag.NodesMax,
			NodesMin:                            r.flag.NodesMin,
			OnDemandBaseCapacity:                r.flag.OnDemandBaseCapacity,
			OnDemandPercentageAboveBaseCapacity: r.flag.OnDemandPercentageAboveBaseCapacity,
			Organization:                        r.flag.Organization,
			ReleaseVersion:                      r.flag.Release,
			UseAlikeInstanceTypes:               r.flag.UseAlikeInstanceTypes,
			EKS:                                 r.flag.EKS,
		}

		if config.NodePoolName == "" {
			generatedName, err := key.GenerateName(true)
			if err != nil {
				return microerror.Mask(err)
			}

			config.NodePoolName = generatedName
		}

		// Remove leading 'v' from release flag input.
		config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

		if len(r.flag.AvailabilityZones) > 0 {
			config.AvailabilityZones = r.flag.AvailabilityZones
		}

		if r.flag.Provider == key.ProviderAzure {
			config.Namespace = key.OrganizationNamespaceFromName(config.Organization)
		}
		if r.flag.Provider == key.ProviderAWS {
			config.Namespace = r.flag.ClusterNamespace
		}
	}

	c, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
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
		err = provider.WriteAWSTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAzure:
		err = provider.WriteAzureTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
