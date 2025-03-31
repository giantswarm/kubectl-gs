package provider

import (
	"context"
	"fmt"
	"io"
	"math"
	"slices"
	"text/template"

	"github.com/3th1nk/cidr"
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/pkg/errors"
	"k8s.io/utils/net"
	capainfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	gsannotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/provider/templates/capa"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v5/pkg/template/app"
)

const (
	DefaultAppsAWSRepoName = "default-apps-aws"
	ClusterAWSRepoName     = "cluster-aws"
	ModePrivate            = "private"
	ProxyPrivateType       = "proxy-private"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	err := templateClusterCAPA(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateClusterCAPA(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	appName := config.Name
	configMapName := common.UserConfigMapName(appName)

	if len(config.AWS.MachinePool.AZs) == 0 {
		config.AWS.MachinePool.AZs = config.ControlPlaneAZ
	}

	var configMapYAML []byte
	{
		flagValues := BuildCapaClusterConfig(config)

		if len(config.AWS.ControlPlaneLoadBalancerIngressAllowCIDRBlocks) > 0 {
			if config.ManagementCluster == "" {
				// Should have been checked in flag validation code
				return errors.New("logic error - ManagementCluster empty")
			}

			managementCluster := &capainfrav1.AWSCluster{}
			err := k8sClient.CtrlClient().Get(ctx, client.ObjectKey{
				Namespace: "org-giantswarm",
				Name:      config.ManagementCluster,
			}, managementCluster)
			if err != nil {
				return errors.Wrap(err, "failed to get management cluster's AWSCluster object")
			}

			if len(managementCluster.Status.Network.NatGatewaysIPs) == 0 {
				return errors.New("management cluster's AWSCluster object did not have the list `.status.networkStatus.natGatewaysIPs` filled yet, cannot determine IP ranges to allowlist")
			}

			for _, ip := range managementCluster.Status.Network.NatGatewaysIPs {
				var cidr string
				if net.IsIPv4String(ip) {
					cidr = ip + "/32"
				} else {
					return fmt.Errorf("management cluster's AWSCluster object had an invalid IPv4 in `.status.networkStatus.natGatewaysIPs`: %q", ip)
				}

				if !slices.Contains(flagValues.Global.ControlPlane.LoadBalancerIngressAllowCIDRBlocks, cidr) {
					flagValues.Global.ControlPlane.LoadBalancerIngressAllowCIDRBlocks = append(flagValues.Global.ControlPlane.LoadBalancerIngressAllowCIDRBlocks, cidr)
				}
			}

			for _, cidr := range config.AWS.ControlPlaneLoadBalancerIngressAllowCIDRBlocks {
				if cidr == "" {
					// We allow specifying an empty value `--control-plane-load-balancer-ingress-allow-cidr-block ""`
					// to denote that only the management cluster's IPs should be allowed. Skip this value.
				} else if net.IsIPv4CIDRString(cidr) {
					flagValues.Global.ControlPlane.LoadBalancerIngressAllowCIDRBlocks = append(flagValues.Global.ControlPlane.LoadBalancerIngressAllowCIDRBlocks, cidr)
				} else {
					return fmt.Errorf("invalid CIDR (for single IPv4, please use `/32` suffix): %q", cidr)
				}
			}
		}

		if config.AWS.NetworkVPCCIDR != "" {
			c, err := cidr.Parse(config.AWS.NetworkVPCCIDR)
			if err != nil {
				return fmt.Errorf("failed to parse VPC CIDR %q: %w", config.AWS.NetworkVPCCIDR, err)
			}

			subnetCount := config.AWS.NetworkAZUsageLimit

			//I cluster has public subnets, first split will be used for the public subnets, the last 3 splits for private subnets
			//if cluster has only private subnets, all 4 splits will be used for private subnets
			cidrSplit, err := c.SubNetting(cidr.MethodSubnetNum, 4)
			if err != nil {
				return fmt.Errorf("failed to split VPC CIDR %q into 4 subnets: %w", config.AWS.NetworkVPCCIDR, err)
			}

			//initialize the subnet structure, there will always be private subnets
			flagValues.Global.Connectivity.Subnets = []capa.Subnet{
				{
					IsPublic:   false,
					CidrBlocks: []capa.CIDRBlock{},
				},
			}

			//if cluster has public subnets, we use splits 1 for public subnets and 2,3,4 for private subnets
			privateCidrSplitStart := 1
			if config.AWS.ClusterType == ProxyPrivateType {
				//if cluster has only private subnets we use all 4 splits
				privateCidrSplitStart = 0
			}

			privateSubnetCount := 0
			// loop over the private subnet splits in order to generate the required amount of private subnets
			for j := privateCidrSplitStart; j < 4 && privateSubnetCount < subnetCount; j++ {
				ones, _ := cidrSplit[j].MaskSize()

				if config.AWS.PrivateSubnetMask < ones {
					return fmt.Errorf("failed to split VPC CIDR %q into subnets because subsplit of private subnet %q failed: you must specify a smaller private subnet size than `/%d` i.e. larger value for --%s)", config.AWS.NetworkVPCCIDR, cidrSplit[j].CIDR(), config.AWS.PrivateSubnetMask, flags.FlagAWSPrivateSubnetMask)
				}

				//divide the current split in blocks with the size of the private subnet mask
				availablePrivateSubnetSplits, err := cidrSplit[j].SubNetting(cidr.MethodSubnetNum, int(math.Pow(2, float64(config.AWS.PrivateSubnetMask-ones))))
				if err != nil {
					return fmt.Errorf("failed to split VPC CIDR %q into subnets because subsplit of private subnet %q failed (using specified private subnet size `/%d`): %w", config.AWS.NetworkVPCCIDR, cidrSplit[j].CIDR(), config.AWS.PrivateSubnetMask, err)
				}

				//while there is space in the current split, generate private subnets
				for k := 0; k < len(availablePrivateSubnetSplits) && privateSubnetCount < subnetCount; k++ {
					flagValues.Global.Connectivity.Subnets[0].CidrBlocks = append(flagValues.Global.Connectivity.Subnets[0].CidrBlocks, capa.CIDRBlock{
						CIDR:             availablePrivateSubnetSplits[k].CIDR().String(),
						AvailabilityZone: string(rune('a' + privateSubnetCount)), // Adjusted to start from 'a' for each subnet
					})
					privateSubnetCount++
				}
			}

			if config.AWS.ClusterType != ProxyPrivateType {

				//initialize public subnets in the cluster structure
				flagValues.Global.Connectivity.Subnets = append(flagValues.Global.Connectivity.Subnets, capa.Subnet{
					IsPublic:   true,
					CidrBlocks: []capa.CIDRBlock{},
				})

				//always use the first split for public subnets
				ones, _ := cidrSplit[0].MaskSize()

				//divide the current split in blocks with the size of the public subnet mask
				availablePublicSubnets, err := cidrSplit[0].SubNetting(cidr.MethodSubnetNum, int(math.Pow(2, float64(config.AWS.PublicSubnetMask-ones))))
				if err != nil {
					return microerror.Mask(err)
				}

				//generate public subnets
				for i := 0; i < subnetCount; i++ {
					flagValues.Global.Connectivity.Subnets[1].CidrBlocks = append(flagValues.Global.Connectivity.Subnets[1].CidrBlocks, capa.CIDRBlock{
						CIDR:             availablePublicSubnets[i].CIDR().String(),
						AvailabilityZone: string(rune('a' + i)), // generate `a`, `b`, etc. based on which index we're at
					})
				}
			}

		}

		if config.AWS.ClusterType == ProxyPrivateType {
			httpProxy := config.AWS.HttpsProxy
			if config.AWS.HttpProxy != "" {
				httpProxy = config.AWS.HttpProxy
			}
			flagValues.Global.Connectivity.Proxy = &capa.Proxy{
				Enabled:    true,
				HttpsProxy: config.AWS.HttpsProxy,
				HttpProxy:  httpProxy,
				NoProxy:    config.AWS.NoProxy,
			}

			flagValues.Global.ControlPlane.APIMode = common.DefaultTo(config.AWS.APIMode, ModePrivate)
			flagValues.Global.Connectivity.VPCMode = common.DefaultTo(config.AWS.VPCMode, ModePrivate)
			flagValues.Global.Connectivity.Topology.Mode = common.DefaultTo(config.AWS.TopologyMode, gsannotation.NetworkTopologyModeGiantSwarmManaged)
			flagValues.Global.Connectivity.Topology.PrefixListID = config.AWS.PrefixListID
			flagValues.Global.Connectivity.Topology.TransitGatewayID = config.AWS.TransitGatewayID
		}

		configData, err := capa.GenerateClusterValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
			Name:      configMapName,
			Namespace: common.OrganizationNamespace(config.Organization),
			Data:      configData,
		})
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap.Labels = map[string]string{}
		userConfigMap.Labels[k8smetadata.Cluster] = config.Name
		if config.PreventDeletion {
			userConfigMap.Labels[label.PreventDeletion] = "true" //nolint:goconst
		}

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appYAML []byte
	{
		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterAWSRepoName,
			Namespace:               common.OrganizationNamespace(config.Organization),
			UserConfigConfigMapName: configMapName,
			ExtraLabels:             map[string]string{},
		}
		if config.PreventDeletion {
			clusterAppConfig.ExtraLabels[label.PreventDeletion] = "true"
		}

		var err error
		appYAML, err = templateapp.NewAppCR(clusterAppConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err := t.Execute(output, templateapp.AppCROutput{
		AppCR:               string(appYAML),
		UserConfigConfigMap: string(configMapYAML),
	})
	return microerror.Mask(err)
}

func BuildCapaClusterConfig(config common.ClusterConfig) capa.ClusterConfig {
	return capa.ClusterConfig{
		Global: &capa.Global{
			Connectivity: &capa.Connectivity{
				AvailabilityZoneUsageLimit: config.AWS.NetworkAZUsageLimit,
				Network: &capa.Network{
					VPCCIDR: config.AWS.NetworkVPCCIDR,
				},
				Topology: &capa.Topology{},
			},
			ControlPlane: &capa.ControlPlane{
				InstanceType: config.ControlPlaneInstanceType,
			},
			Metadata: &capa.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
			},
			NodePools: &map[string]capa.MachinePool{
				config.AWS.MachinePool.Name: {
					AvailabilityZones: config.AWS.MachinePool.AZs,
					InstanceType:      config.AWS.MachinePool.InstanceType,
					MinSize:           config.AWS.MachinePool.MinSize,
					MaxSize:           config.AWS.MachinePool.MaxSize,
					RootVolumeSizeGB:  config.AWS.MachinePool.RootVolumeSizeGB,
					CustomNodeLabels:  config.AWS.MachinePool.CustomNodeLabels,
				},
			},
			ProviderSpecific: &capa.ProviderSpecific{
				Region:                     config.Region,
				AWSClusterRoleIdentityName: config.AWS.AWSClusterRoleIdentityName,
			},
			Release: &capa.Release{
				Version: config.ReleaseVersion,
			},
		},
	}
}
