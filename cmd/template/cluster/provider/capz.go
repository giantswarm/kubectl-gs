package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v6/pkg/template/app"
)

const (
	ClusterAzureRepoName = "cluster-azure"
	ReleaseAzureRepoName = "release-azure"
)

func WriteCAPZTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	err := templateClusterCAPZ(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateClusterCAPZ(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	appName := config.Name
	configMapName := common.UserConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapzClusterConfig(config)

		// For release versions, the release version is baked into the chart,
		// so we don't need to include it in the user config.
		if config.UseReleaseChart {
			flagValues.Global.Release = nil
		}

		configData, err := capz.GenerateClusterValues(flagValues)
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
		ociRepoYAML, helmReleaseYAML, err := common.BuildClusterFluxResources(config, ReleaseAzureRepoName, configMapName)
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
		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterAzureRepoName,
			Namespace:               common.OrganizationNamespace(config.Organization),
			UserConfigConfigMapName: configMapName,
			ExtraLabels:             map[string]string{},
			// Only an explicitly requested version is set; otherwise the
			// app-operator webhook resolves the version from the Release CR.
			Version: config.App.ClusterVersion,
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

func BuildCapzClusterConfig(config common.ClusterConfig) capz.ClusterConfig {
	providerSpecific := &capz.ProviderSpecific{
		Location:       config.Region,
		SubscriptionID: config.Azure.SubscriptionID,
	}
	// Only emit the azureClusterIdentity block when the user overrode the reference,
	// otherwise rely on the chart's defaults.
	if config.Azure.ClusterIdentityName != "" || config.Azure.ClusterIdentityNamespace != "" {
		providerSpecific.AzureClusterIdentity = &capz.AzureClusterIdentity{
			Name:      config.Azure.ClusterIdentityName,
			Namespace: config.Azure.ClusterIdentityNamespace,
		}
	}

	return capz.ClusterConfig{
		Global: &capz.Global{
			Metadata: &capz.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Labels:          config.Labels,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
				ServicePriority: config.ServicePriority,
			},
			ProviderSpecific: providerSpecific,
			Connectivity: &capz.Connectivity{
				Bastion: &capz.Bastion{
					Enabled:      true,
					InstanceType: config.BastionInstanceType,
				},
			},
			ControlPlane: &capz.ControlPlane{
				InstanceType: config.ControlPlaneInstanceType,
				Replicas:     3,
			},
			Release: &capz.Release{
				Version: config.ReleaseVersion,
			},
		},
	}
}
