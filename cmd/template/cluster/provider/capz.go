package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v4/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/v4/pkg/template/app"
)

const (
	ClusterAzureRepoName = "cluster-azure"
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
		userConfigMap.Labels[k8smetadata.Cluster] = config.Name

		configMapYAML, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}
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
