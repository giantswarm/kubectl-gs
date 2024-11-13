package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"

	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/provider/templates/capvcd"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v5/pkg/template/app"
)

const (
	ClusterCloudDirectorRepoName = "cluster-cloud-director"
)

func WriteCloudDirectorTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	appVersion := config.App.ClusterVersion
	if appVersion == "" {
		var err error
		appVersion, err = common.GetLatestVersion(ctx, client.CtrlClient(), ClusterCloudDirectorRepoName, config.App.ClusterCatalog)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err := templateClusterCloudDirector(output, config, appVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateClusterCloudDirector(output io.Writer, config common.ClusterConfig, appVersion string) error {
	appName := config.Name
	configMapName := common.UserConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapvcdClusterConfig(config)

		configData, err := capvcd.GenerateClusterValues(flagValues)
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

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
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
			Name:                    ClusterCloudDirectorRepoName,
			Namespace:               common.OrganizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
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

func BuildCapvcdClusterConfig(config common.ClusterConfig) capvcd.ClusterConfig {
	cfg := capvcd.ClusterConfig{
		Global: &capvcd.Global{
			Connectivity: &capvcd.Connectivity{
				BaseDomain: "test.gigantic.io",
				Network: &capvcd.Network{
					LoadBalancers: &capvcd.LoadBalancers{
						VipSubnet: config.CloudDirector.VipSubnet,
					},
				},
			},
			ControlPlane: &capvcd.ControlPlane{
				Replicas: config.CloudDirector.ControlPlane.Replicas,
				Oidc: &common.OIDC{
					IssuerURL:     config.OIDC.IssuerURL,
					ClientID:      config.OIDC.ClientID,
					UsernameClaim: config.OIDC.UsernameClaim,
					GroupsClaim:   config.OIDC.GroupsClaim,
				},
				MachineTemplate: getCapvdMachineTemplate(&config.CloudDirector.ControlPlane.MachineTemplate),
			},
			Metadata: &capvcd.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
			},
			NodePools: map[string]*capvcd.NodePool{
				"worker": {
					Catalog:      "giantswarm",
					DiskSizeGB:   config.CloudDirector.Worker.DiskSizeGB,
					SizingPolicy: config.CloudDirector.Worker.SizingPolicy,
					Replicas:     config.CloudDirector.Worker.Replicas,
				},
			},
			Release: &capvcd.Release{
				Version: config.ReleaseVersion,
			},
			ProviderSpecific: &capvcd.ProviderSpecific{
				Org:         config.CloudDirector.Org,
				Ovdc:        config.CloudDirector.Ovdc,
				OvdcNetwork: config.CloudDirector.OvdcNetwork,
				Site:        config.CloudDirector.Site,
				UserContext: &capvcd.UserContext{
					SecretRef: &capvcd.SecretRef{
						SecretName: config.CloudDirector.CredentialsSecretName,
					},
				},
			},
		},
	}
	if config.CloudDirector.HttpProxy != "" && config.CloudDirector.HttpsProxy != "" && config.CloudDirector.NoProxy != "" {
		cfg.Global.Connectivity.Proxy = &capvcd.Proxy{
			Enabled:    true,
			HttpsProxy: config.CloudDirector.HttpsProxy,
			HttpProxy:  config.CloudDirector.HttpProxy,
			NoProxy:    config.CloudDirector.NoProxy,
		}
	}

	return cfg
}

func getCapvdMachineTemplate(machineTemplate *common.CloudDirectorMachineTemplate) *capvcd.MachineTemplate {
	return &capvcd.MachineTemplate{
		Catalog:      "giantswarm",
		DiskSizeGB:   machineTemplate.DiskSizeGB,
		SizingPolicy: machineTemplate.SizingPolicy,
	}
}
