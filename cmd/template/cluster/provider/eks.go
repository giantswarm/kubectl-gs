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

	"github.com/giantswarm/kubectl-gs/v3/cmd/template/cluster/provider/templates/capa"
	"github.com/giantswarm/kubectl-gs/v3/cmd/template/cluster/provider/templates/eks"
	"github.com/giantswarm/kubectl-gs/v3/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v3/pkg/template/app"
)

const (
	DefaultAppsEKSRepoName = "default-apps-eks"
	ClusterEKSRepoName     = "cluster-eks"
)

func WriteEKSTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	err := templateClusterEKS(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsEKS(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateClusterEKS(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := config.Name
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildEKSClusterConfig(config)
		configData, err := eks.GenerateClusterValues(flagValues)
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), ClusterEKSRepoName, config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterEKSRepoName,
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

func BuildEKSClusterConfig(config ClusterConfig) eks.ClusterConfig {
	return eks.ClusterConfig{
		Global: &eks.Global{
			Metadata: &eks.Metadata{
				Name:            config.Name,
				Description:     config.Description,
				Organization:    config.Organization,
				PreventDeletion: config.PreventDeletion,
			},
		},
	}
}

func templateDefaultAppsEKS(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsEKSRepoName, config.App.DefaultAppsCatalog)
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
			Name:                    DefaultAppsEKSRepoName,
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
