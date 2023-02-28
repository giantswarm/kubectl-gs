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

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/capa"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
)

const (
	DefaultAppsRepoName = "default-apps-aws"
	ClusterAWSRepoName  = "cluster-aws"
	ModePrivate         = "private"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	var err error

	if config.AWS.EKS {
		err = WriteCAPAEKSTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		err = templateClusterAWS(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}

		err = templateDefaultAppsAWS(ctx, client, output, config)
		return microerror.Mask(err)
	}

	return nil
}

func WriteCAPAEKSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterConfig) error {
	var err error

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		ReleaseVersion    string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.21",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Organization),
		Organization:      config.Organization,
		ReleaseVersion:    config.ReleaseVersion,
	}

	var templates []templateConfig
	for _, t := range aws.GetEKSTemplates() {
		templates = append(templates, templateConfig(t))
	}

	err = runMutation(ctx, client, data, templates, out)
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

			flagValues.Network.Subnets = []capa.Subnet{
				{
					CidrBlocks: []capa.CIDRBlock{},
				},
			}

			for i := 0; i < subnetCount; i++ {
				flagValues.Network.Subnets[0].CidrBlocks = append(flagValues.Network.Subnets[0].CidrBlocks, capa.CIDRBlock{
					CIDR:             subnets[i].CIDR().String(),
					AvailabilityZone: string(rune('a' + i)), // generate `a`, `b`, etc. based on which index we're at
				})
			}

			httpProxy := config.AWS.HttpsProxy
			if config.AWS.HttpProxy != "" {
				httpProxy = config.AWS.HttpProxy
			}
			flagValues.Proxy = &capa.Proxy{
				Enabled:    true,
				HttpsProxy: config.AWS.HttpsProxy,
				HttpProxy:  httpProxy,
				NoProxy:    config.AWS.NoProxy,
			}

			flagValues.Network.APIMode = defaultTo(config.AWS.APIMode, ModePrivate)
			flagValues.Network.VPCMode = defaultTo(config.AWS.VPCMode, ModePrivate)
			flagValues.Network.DNSMode = defaultTo(config.AWS.DNSMode, ModePrivate)
			flagValues.Network.TopologyMode = defaultTo(config.AWS.TopologyMode, gsannotation.NetworkTopologyModeGiantSwarmManaged)
			flagValues.Network.PrefixListID = config.AWS.PrefixListID
			flagValues.Network.TransitGatewayID = config.AWS.TransitGatewayID
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
		ClusterName:        config.Name,
		ClusterDescription: config.Description,
		Organization:       config.Organization,

		AWS: &capa.AWS{
			Region:                     config.Region,
			AWSClusterRoleIdentityName: config.AWS.AWSClusterRoleIdentityName,
		},
		Network: &capa.Network{
			AvailabilityZoneUsageLimit: config.AWS.NetworkAZUsageLimit,
			VPCCIDR:                    config.AWS.NetworkVPCCIDR,
		},
		Bastion: &capa.Bastion{
			InstanceType: config.BastionInstanceType,
			Replicas:     config.BastionReplicas,
		},
		ControlPlane: &capa.ControlPlane{
			InstanceType: config.ControlPlaneInstanceType,
			Replicas:     3,
		},
		MachinePools: &map[string]capa.MachinePool{
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsRepoName, config.App.DefaultAppsCatalog)
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
			Name:                    DefaultAppsRepoName,
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
