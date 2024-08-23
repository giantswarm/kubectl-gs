package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"
	capg "github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/provider/templates/gcp"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v5/pkg/template/app"
)

const (
	DefaultAppsGCPRepoName = "default-apps-gcp"
	ClusterGCPRepoName     = "cluster-gcp"
)

func WriteGCPTemplate(ctx context.Context, client k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	err := templateClusterGCP(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsGCP(ctx, client, output, config)
	return microerror.Mask(err)
}

func templateClusterGCP(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	appName := config.Name
	configMapName := common.UserConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := BuildCapgClusterConfig(config)

		configData, err := capg.GenerateClusterValues(flagValues)
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
		appVersion := config.App.ClusterVersion
		if appVersion == "" {
			var err error
			appVersion, err = common.GetLatestVersion(ctx, k8sClient.CtrlClient(), ClusterGCPRepoName, config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 config.Name,
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    ClusterGCPRepoName,
			Namespace:               common.OrganizationNamespace(config.Organization),
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

func BuildCapgClusterConfig(config common.ClusterConfig) capg.ClusterConfig {
	return capg.ClusterConfig{
		ClusterName:        config.Name,
		ClusterDescription: config.Description,
		Organization:       config.Organization,

		GCP: &capg.GCP{
			Region:         config.Region,
			Project:        config.GCP.Project,
			FailureDomains: config.GCP.FailureDomains,
		},
		BastionInstanceType: config.BastionInstanceType,
		ControlPlane: &capg.ControlPlane{
			InstanceType: config.ControlPlaneInstanceType,
			Replicas:     3,
			ServiceAccount: capg.ServiceAccount{
				Email:  config.GCP.ControlPlane.ServiceAccount.Email,
				Scopes: config.GCP.ControlPlane.ServiceAccount.Scopes,
			},
		},
		MachineDeployments: &[]capg.MachineDeployment{
			{
				Name:          config.GCP.MachineDeployment.Name,
				FailureDomain: config.GCP.MachineDeployment.FailureDomain,
				InstanceType:  config.GCP.MachineDeployment.InstanceType,
				Replicas:      config.GCP.MachineDeployment.Replicas,
				RootVolume: capg.Volume{
					SizeGB: config.GCP.MachineDeployment.RootVolumeSizeGB,
				},
				CustomNodeLabels: config.GCP.MachineDeployment.CustomNodeLabels,
				ServiceAccount: capg.ServiceAccount{
					Email:  config.GCP.MachineDeployment.ServiceAccount.Email,
					Scopes: config.GCP.MachineDeployment.ServiceAccount.Scopes,
				},
			},
		},
	}
}

func templateDefaultAppsGCP(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config common.ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := common.UserConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := capg.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		configData, err := capg.GenerateDefaultAppsValues(flagValues)
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
		appVersion := config.App.DefaultAppsVersion
		if appVersion == "" {
			var err error
			appVersion, err = common.GetLatestVersion(ctx, k8sClient.CtrlClient(), DefaultAppsGCPRepoName, config.App.DefaultAppsCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		var err error
		appYAML, err = templateapp.NewAppCR(templateapp.Config{
			AppName:                 appName,
			Cluster:                 config.Name,
			Catalog:                 config.App.DefaultAppsCatalog,
			InCluster:               true,
			Name:                    DefaultAppsGCPRepoName,
			Namespace:               common.OrganizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
			DefaultingEnabled:       false,
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
