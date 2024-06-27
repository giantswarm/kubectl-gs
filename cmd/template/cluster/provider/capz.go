package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/v3/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v3/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v3/pkg/template/app"
)

const (
	DefaultAppsAzureRepoName = "default-apps-azure"
	ClusterAzureRepoName     = "cluster-azure"
)

func WriteCAPZTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appVersion := config.App.ClusterVersion
	if appVersion == "" {
		var err error
		appVersion, err = getLatestVersion(ctx, client.CtrlClient(), ClusterAzureRepoName, config.App.ClusterCatalog)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err := templateClusterCAPZ(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	minUnifiedClusterAzureVersion := semver.New(0, 14, 0, "", "")
	desiredClusterAzureVersion, err := semver.StrictNewVersion(appVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if desiredClusterAzureVersion.LessThan(minUnifiedClusterAzureVersion) {
		// Render default-apps-azure only when cluster-azure version does not contain default apps.
		err = templateDefaultAppsAzure(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func templateClusterCAPZ(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := config.Name
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapzClusterConfig(config)

		configData, err := capz.GenerateClusterValues(flagValues)
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), ClusterAzureRepoName, config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterAzureRepoName,
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

func BuildCapzClusterConfig(config ClusterConfig) capz.ClusterConfig {
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
		},
	}
}

func templateDefaultAppsAzure(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := capz.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		configData, err := capz.GenerateDefaultAppsValues(flagValues)
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsAzureRepoName, config.App.DefaultAppsCatalog)
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
			Name:                    DefaultAppsAzureRepoName,
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
