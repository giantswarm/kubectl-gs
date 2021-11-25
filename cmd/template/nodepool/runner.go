package nodepool

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/giantswarm/apiextensions/v3/pkg/id"
	"github.com/giantswarm/apiextensions/v3/pkg/label"

	"github.com/giantswarm/kubectl-gs/cmd/template/nodepool/provider"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

const (
	nodePoolCRFileName = "nodepoolCR"
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

	commonConfig := commonconfig.New(r.flag.config)
	c, err := commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

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

		if config.NodePoolID == "" {
			config.NodePoolID = id.Generate()
		}

		if len(r.flag.AvailabilityZones) > 0 {
			config.AvailabilityZones = r.flag.AvailabilityZones
		}

		if r.flag.Provider == key.ProviderAzure {
			config.Namespace = key.OrganizationNamespaceFromName(config.Organization)
		}
		if r.flag.Provider == key.ProviderAWS {
			config.Namespace = r.flag.ClusterNamespace
		}

		var clusterService cluster.Interface
		{
			serviceConfig := cluster.Config{
				Client: c,
			}
			clusterService, err = cluster.New(serviceConfig)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		wc, err := getWorkloadCluster(ctx, clusterService, config.Namespace, config.ClusterName, r.flag.Provider)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(config.ReleaseVersion) < 1 {
			config.ReleaseVersion = wc.Cluster.Labels[label.ReleaseVersion]
		} else {
			config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")
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
		err = provider.WriteAWSTemplate(ctx, c.K8sClient, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAzure:
		err = provider.WriteAzureTemplate(ctx, c.K8sClient, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func getWorkloadCluster(ctx context.Context, clusterService cluster.Interface, namespace, name, provider string) (*cluster.Cluster, error) {
	o := cluster.GetOptions{
		Namespace: namespace,
		Name:      name,
		Provider:  provider,
	}

	c, err := clusterService.Get(ctx, o)
	if cluster.IsNotFound(err) || cluster.IsNoResources(err) {
		return nil, microerror.Maskf(clusterNotFoundError, "The workload cluster %s could not be found.", name)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	switch wc := c.(type) {
	case *cluster.Cluster:
		return wc, nil
	case *cluster.Collection:
		targetWC := wc.Items[0]
		return &targetWC, nil
	default:
		return nil, microerror.Maskf(clusterNotFoundError, "The workload cluster %s could not be found.", name)
	}
}
