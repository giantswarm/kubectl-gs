package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider/templates/capv"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v6/pkg/template/app"
)

const (
	ClusterVsphereRepoName = "cluster-vsphere"
	ReleaseVsphereRepoName = "release-vsphere"
)

func WriteVSphereTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	// Only an explicitly requested version is set; otherwise the
	// app-operator webhook resolves the version from the Release CR. Not
	// needed at all on the release-<provider> chart path.
	appVersion := config.App.ClusterVersion
	if !config.UseReleaseChart && appVersion == "" {
		var err error
		appVersion, err = common.GetLatestVersion(ctx, client.CtrlClient(), ClusterVsphereRepoName, config.App.ClusterCatalog)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return templateClusterVSphere(output, config, appVersion)
}

func templateClusterVSphere(output io.Writer, config common.ClusterConfig, appVersion string) error {
	appName := config.Name
	configMapName := common.UserConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapvClusterConfig(config)

		// For release versions, the release version is baked into the chart,
		// so we don't need to include it in the user config.
		if config.UseReleaseChart {
			flagValues.Global.Release = nil
		}

		configData, err := capv.GenerateClusterValues(flagValues)
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
		userConfigMap.Labels[label.Cluster] = config.Name
		for k, v := range config.Labels {
			userConfigMap.Labels[k] = v
		}
		if config.PreventDeletion {
			userConfigMap.Labels[label.PreventDeletion] = "true" //nolint:goconst
		}

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if config.UseReleaseChart {
		ociRepoYAML, helmReleaseYAML, err := common.BuildClusterFluxResources(config, ReleaseVsphereRepoName, configMapName)
		if err != nil {
			return microerror.Mask(err)
		}

		t := template.Must(template.New("clusterFlux").Parse(key.ClusterFluxTemplate))
		return microerror.Mask(t.Execute(output, templateapp.ClusterFluxOutput{
			UserConfigConfigMap: string(configMapYAML),
			OCIRepository:       string(ociRepoYAML),
			HelmRelease:         string(helmReleaseYAML),
		}))
	}

	var appYAML []byte
	{
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
			Namespace:               common.OrganizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
			UserConfigSecretName:    config.VSphere.CredentialsSecretName,
			ExtraConfigs:            extraConfigs,
			ExtraLabels:             map[string]string{},
		}
		for k, v := range config.Labels {
			clusterAppConfig.ExtraLabels[k] = v
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

func BuildCapvClusterConfig(config common.ClusterConfig) capv.ClusterConfig {
	cfg := capv.ClusterConfig{
		Global: &capv.Global{
			Connectivity: &capv.Connectivity{
				BaseDomain: "test.gigantic.io",
				Network: &capv.Network{
					ControlPlaneEndpoint: &capv.ControlPlaneEndpoint{
						Host:       config.VSphere.ControlPlane.Ip,
						IpPoolName: config.VSphere.ControlPlane.IpPoolName,
						Port:       6443,
					},
					LoadBalancers: &capv.LoadBalancers{
						IpPoolName: config.VSphere.SvcLbIpPoolName,
					},
				},
			},
			ControlPlane: &capv.ControlPlane{
				Replicas:        config.VSphere.ControlPlane.Replicas,
				MachineTemplate: getMachineTemplate(&config.VSphere.ControlPlane.VSphereMachineTemplate, &config),
			},
			Metadata: &capv.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Labels:          config.Labels,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
				ServicePriority: config.ServicePriority,
			},
			NodePools: map[string]*capv.NodePool{
				"worker": getNodePool(getMachineTemplate(&config.VSphere.Worker, &config), config.VSphere.Worker.Replicas),
			},
			Release: &capv.Release{
				Version: config.ReleaseVersion,
			},
		},
	}
	if config.VSphere.ServiceLoadBalancerCIDR != "" {
		cfg.Global.Connectivity.Network.LoadBalancers.CidrBlocks = []string{config.VSphere.ServiceLoadBalancerCIDR}
	}
	return cfg
}

func getMachineTemplate(machineTemplate *common.VSphereMachineTemplate, clusterConfig *common.ClusterConfig) *capv.MachineTemplate {
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
	}
}

func getNodePool(machineTemplate *capv.MachineTemplate, replicas int) *capv.NodePool {
	return &capv.NodePool{
		Replicas:     replicas,
		Network:      machineTemplate.Network,
		CloneMode:    machineTemplate.CloneMode,
		DiskGiB:      machineTemplate.DiskGiB,
		NumCPUs:      machineTemplate.NumCPUs,
		MemoryMiB:    machineTemplate.MemoryMiB,
		ResourcePool: machineTemplate.ResourcePool,
		Template:     machineTemplate.Template,
	}
}
