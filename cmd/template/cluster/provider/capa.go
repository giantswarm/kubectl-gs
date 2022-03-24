package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/capa"
	"github.com/giantswarm/kubectl-gs/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
)

const (
	DefaultAppsRepoName = "default-apps-aws"
	ClusterAWSRepoName  = "cluster-aws"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, output *os.File, config ClusterConfig) error {
	var err error

	var sshSSOPublicKey string
	{
		sshSSOPublicKey, err = key.SSHSSOPublicKey(ctx, client.CtrlClient())
		if err != nil {
			return microerror.Mask(err)
		}
	}
	config.AWS.SSHSSOPublicKey = sshSSOPublicKey

	err = templateClusterAWS(ctx, client, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsAWS(ctx, client, output, config)
	return microerror.Mask(err)

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

func templateClusterAWS(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterConfig) error {
	appName := config.Name
	configMapName := userConfigMapName(appName)

	var configMapYAML []byte
	{
		flagValues := capa.ClusterConfig{
			ClusterDescription: config.Description,
			Organization:       config.Organization,

			AWS: &capa.AWS{
				Region: config.AWS.Region,
				Role:   config.AWS.Role,
			},
			Network: &capa.Network{
				AvailabilityZoneUsageLimit: config.AWS.NetworkAZUsageLimit,
				VPCCIDR:                    config.AWS.NetworkVPCCIDR,
			},
			Bastion: &capa.Bastion{
				InstanceType: config.AWS.BastionInstanceType,
				Replicas:     config.AWS.BastionReplicas,
			},
			ControlPlane: &capa.ControlPlane{
				InstanceType:      config.AWS.ControlPlaneInstanceType,
				Replicas:          3,
				AvailabilityZones: config.ControlPlaneAZ,
			},
			MachinePools: &[]capa.MachinePool{
				{
					Name:              config.AWS.MachinePool.Name,
					AvailabilityZones: config.AWS.MachinePool.AZs,
					InstanceType:      config.AWS.MachinePool.InstanceType,
					MinSize:           config.AWS.MachinePool.MinSize,
					MaxSize:           config.AWS.MachinePool.MaxSize,
					RootVolumeSizeGB:  config.AWS.MachinePool.RootVolumeSizeGB,
					CustomNodeLabels:  config.AWS.MachinePool.CustomNodeLabels,
				},
			},
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

		userConfigMap.ObjectMeta.Labels = map[string]string{}
		userConfigMap.ObjectMeta.Labels[k8smetadata.Cluster] = config.Name

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
			ExtraLabels: map[string]string{
				k8smetadata.Cluster: config.Name,
			},
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

func templateDefaultAppsAWS(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterConfig) error {
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

		userConfigMap.ObjectMeta.Labels = map[string]string{}
		userConfigMap.ObjectMeta.Labels[k8smetadata.Cluster] = config.Name

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
			Catalog:                 config.App.DefaultAppsCatalog,
			InCluster:               true,
			Name:                    DefaultAppsRepoName,
			Namespace:               organizationNamespace(config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
			ExtraLabels: map[string]string{
				k8smetadata.Cluster: config.Name,
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
