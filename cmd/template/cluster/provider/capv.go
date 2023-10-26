package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/capv"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
)

const (
	DefaultAppsVsphereRepoName = "default-apps-vsphere"
	ClusterVsphereRepoName     = "cluster-vsphere"
)

func WriteVSphereTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	err := templateClusterVSphere(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsVsphere(ctx, client, output, config)
	return microerror.Mask(err)
}

func templateClusterVSphere(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := config.Name
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapvClusterConfig(config)

		configData, err := capv.GenerateClusterValues(flagValues)
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), ClusterVsphereRepoName, config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}
		extraConfigs := []applicationv1alpha1.AppExtraConfig{
			{
				Kind:      "secret",
				Name:      "container-registries-configuration",
				Namespace: "default",
				Priority:  25,
			},
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterVsphereRepoName,
			Namespace:               organizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
			UserConfigSecretName:    config.VSphere.CredentialsSecretName,
			ExtraConfigs:            extraConfigs,
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

func BuildCapvClusterConfig(config ClusterConfig) capv.ClusterConfig {
	const className = "default"
	return capv.ClusterConfig{
		BaseDomain:         "test.gigantic.io",
		ClusterDescription: config.Description,
		Organization:       config.Organization,
		Cluster: &capv.Cluster{
			KubernetesVersion:        config.KubernetesVersion,
			EnableEncryptionProvider: false,
		},
		Connectivity: &capv.Connectivity{
			Network: &capv.Network{
				AllowAllEgress: true,
				ControlPlaneEndpoint: &capv.ControlPlaneEndpoint{
					Host:       config.VSphere.ControlPlane.IP,
					IpPoolName: config.VSphere.ControlPlane.IPPoolName,
					Port:       6443,
				},
				LoadBalancers: &capv.LoadBalancers{
					CidrBlocks: []string{
						config.VSphere.ServiceLoadBalancerCIDR,
					},
				},
			},
		},
		ControlPlane: &capv.ControlPlane{
			Replicas: config.VSphere.ControlPlane.Replicas,
			Image: &capv.Image{
				Repository: "registry.k8s.io",
			},
			MachineTemplate: getMachineTemplate(&config.VSphere.ControlPlane.VSphereMachineTemplate, &config),
		},
		NodeClasses: map[string]*capv.MachineTemplate{
			className: getMachineTemplate(&config.VSphere.Worker, &config),
		},
		NodePools: map[string]*capv.NodePool{
			"worker": {
				Class:    className,
				Replicas: config.VSphere.Worker.Replicas,
			},
		},
		HelmReleases: &capv.HelmReleases{
			Cilium: &capv.HelmRelease{
				Interval: "20s",
			},
			Cpi: &capv.HelmRelease{
				Interval: "30s",
			},
			Coredns: &capv.HelmRelease{
				Interval: "30s",
			},
		},
	}
}

func getMachineTemplate(machineTemplate *VSphereMachineTemplate, clusterConfig *ClusterConfig) *capv.MachineTemplate {
	config := clusterConfig.VSphere
	commonNetwork := &capv.MTNetwork{
		Devices: []*capv.MTDevice{
			{
				NetworkName: config.NetworkName,
				Dhcp4:       true,
			},
		},
	}
	return &capv.MachineTemplate{
		Network:      commonNetwork,
		CloneMode:    "linkedClone",
		DiskGiB:      machineTemplate.DiskGiB,
		NumCPUs:      machineTemplate.NumCPUs,
		MemoryMiB:    machineTemplate.MemoryMiB,
		ResourcePool: config.ResourcePool,
		Template:     config.ImageTemplate,
	}
}

func templateDefaultAppsVsphere(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := capv.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		configData, err := capv.GenerateDefaultAppsValues(flagValues)
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsVsphereRepoName, config.App.DefaultAppsCatalog)
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
			Name:                    DefaultAppsVsphereRepoName,
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
