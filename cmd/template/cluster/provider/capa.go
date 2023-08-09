package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/3th1nk/cidr"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	gsannotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/capa"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
)

const (
	DefaultAppsAWSRepoName = "default-apps-aws"
	ClusterAWSRepoName     = "cluster-aws"
	ModePrivate            = "private"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	err := templateClusterAWS(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsAWS(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateClusterAWS(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := config.Name
	configMapName := userConfigMapName(appName)

	if config.AWS.MachinePool.AZs == nil || len(config.AWS.MachinePool.AZs) == 0 {
		config.AWS.MachinePool.AZs = config.ControlPlaneAZ
	}

	var configMapYAML []byte
	{
		flagValues := BuildCapaClusterConfig(config)

		if config.AWS.ClusterType == "proxy-private" {
			subnetCountLimit := 0
			subnetCount := len(config.AWS.MachinePool.AZs)
			if subnetCount == 0 {
				subnetCountLimit = 4
				subnetCount = config.AWS.NetworkAZUsageLimit
			} else {
				subnetCountLimit = findNextPowerOfTwo(subnetCount)
			}

			c, _ := cidr.Parse(config.AWS.NetworkVPCCIDR)
			subnets, err := c.SubNetting(cidr.MethodSubnetNum, subnetCountLimit)
			if err != nil {
				return microerror.Mask(err)
			}

			flagValues.Connectivity.Subnets = []capa.Subnet{
				{
					CidrBlocks: []capa.CIDRBlock{},
				},
			}

			for i := 0; i < subnetCount; i++ {
				flagValues.Connectivity.Subnets[0].CidrBlocks = append(flagValues.Connectivity.Subnets[0].CidrBlocks, capa.CIDRBlock{
					CIDR:             subnets[i].CIDR().String(),
					AvailabilityZone: string(rune('a' + i)), // generate `a`, `b`, etc. based on which index we're at
				})
			}

			httpProxy := config.AWS.HttpsProxy
			if config.AWS.HttpProxy != "" {
				httpProxy = config.AWS.HttpProxy
			}
			flagValues.Connectivity.Proxy = &capa.Proxy{
				Enabled:    true,
				HttpsProxy: config.AWS.HttpsProxy,
				HttpProxy:  httpProxy,
				NoProxy:    config.AWS.NoProxy,
			}

			flagValues.ControlPlane.APIMode = defaultTo(config.AWS.APIMode, ModePrivate)
			flagValues.Connectivity.VPCMode = defaultTo(config.AWS.VPCMode, ModePrivate)
			flagValues.Connectivity.DNS.Mode = defaultTo(config.AWS.DNSMode, ModePrivate)
			flagValues.Connectivity.Topology.Mode = defaultTo(config.AWS.TopologyMode, gsannotation.NetworkTopologyModeGiantSwarmManaged)
			flagValues.Connectivity.Topology.PrefixListID = config.AWS.PrefixListID
			flagValues.Connectivity.Topology.TransitGatewayID = config.AWS.TransitGatewayID
		}

		configData, err := capa.GenerateClusterValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
			Name:      configMapName,
			Namespace: organizationNamespace(config.Organization),
			Data:      configData,
		})
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap.Labels = map[string]string{}
		userConfigMap.Labels[k8smetadata.Cluster] = config.Name

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appYAML []byte
	{
		appVersion := config.App.ClusterVersion
		if appVersion == "" {
			var err error
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), ClusterAWSRepoName, config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterAWSRepoName,
			Namespace:               organizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
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

func BuildCapaClusterConfig(config ClusterConfig) capa.ClusterConfig {
	return capa.ClusterConfig{
		Metadata: &capa.Metadata{
			Name:         config.Name,
			Description:  config.Description,
			Organization: config.Organization,
		},
		ProviderSpecific: &capa.ProviderSpecific{
			Region:                     config.Region,
			AWSClusterRoleIdentityName: config.AWS.AWSClusterRoleIdentityName,
		},
		Connectivity: &capa.Connectivity{
			AvailabilityZoneUsageLimit: config.AWS.NetworkAZUsageLimit,
			Bastion: &capa.Bastion{
				Enabled:      true,
				InstanceType: config.BastionInstanceType,
				Replicas:     config.BastionReplicas,
			},
			DNS: &capa.DNS{},
			Network: &capa.Network{
				VPCCIDR: config.AWS.NetworkVPCCIDR,
			},
			Topology: &capa.Topology{},
		},
		ControlPlane: &capa.ControlPlane{
			InstanceType: config.ControlPlaneInstanceType,
			Replicas:     3,
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
	}
}

func templateDefaultAppsAWS(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := capa.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		configData, err := capa.GenerateDefaultAppsValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
			Name:      configMapName,
			Namespace: organizationNamespace(config.Organization),
			Data:      configData,
		})
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap.Labels = map[string]string{}
		userConfigMap.Labels[k8smetadata.Cluster] = config.Name

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appYAML []byte
	{
		appVersion := config.App.DefaultAppsVersion
		if appVersion == "" {
			var err error
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsAWSRepoName, config.App.DefaultAppsCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		var err error
		appYAML, err = templateapp.NewAppCR(templateapp.Config{
			AppName:                 appName,
			Cluster:                 config.Name,
			Catalog:                 config.App.DefaultAppsCatalog,
			DefaultingEnabled:       false,
			InCluster:               true,
			Name:                    DefaultAppsAWSRepoName,
			Namespace:               organizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
			UseClusterValuesConfig:  true,
			ExtraLabels: map[string]string{
				k8smetadata.ManagedBy: "cluster",
			},
		})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err := t.Execute(output, templateapp.AppCROutput{
		UserConfigConfigMap: string(configMapYAML),
		AppCR:               string(appYAML),
	})
	return microerror.Mask(err)
}
