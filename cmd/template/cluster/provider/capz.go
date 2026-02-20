package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v5/pkg/template/app"
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
		if common.IsReleaseVersion(config.ReleaseVersion) {
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
		// Use release-<provider> chart name for release versions (>= 35.0.0).
		// These charts have the release version baked into values.yaml.
		// For older chart versions, use cluster-<provider> and let the webhook handle version.
		chartName := ClusterAzureRepoName
		if common.IsReleaseVersion(config.ReleaseVersion) {
			chartName = ReleaseAzureRepoName
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    chartName,
			Namespace:               common.OrganizationNamespace(config.Organization),
			UserConfigConfigMapName: configMapName,
			ExtraLabels:             map[string]string{},
		}
		// Set version for release charts where the chart version equals the release version.
		// For cluster-<provider> charts, the webhook handles version mutation.
		if common.IsReleaseVersion(config.ReleaseVersion) {
			clusterAppConfig.Version = config.ReleaseVersion
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
	return capz.ClusterConfig{
		Global: &capz.Global{
			Metadata: &capz.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
			},
			ProviderSpecific: &capz.ProviderSpecific{
				Location:       config.Region,
				SubscriptionID: config.Azure.SubscriptionID,
			},
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
