package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
)

func WriteOpenStackTemplate(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	err := templateClusterOpenstack(ctx, k8sClient, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsOpenstack(ctx, k8sClient, output, config)
	return microerror.Mask(err)
}

func templateClusterOpenstack(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := config.Name
	configMapName := fmt.Sprintf("%s-cluster-userconfig", appName)

	controlPlaneReplicas := 1
	if len(config.ControlPlaneAZ) > 0 {
		controlPlaneReplicas = len(config.ControlPlaneAZ)
	}

	var configMapYAML []byte
	{
		flagValues := openstack.ClusterConfig{
			ClusterDescription: config.Description,
			ClusterName:        config.Name,
			DNSNameservers:     config.OpenStack.DNSNameservers,
			KubernetesVersion:  config.KubernetesVersion,
			Organization:       config.Organization,
			CloudConfig:        config.OpenStack.CloudConfig,
			CloudName:          config.OpenStack.Cloud,
			NodeCIDR:           config.OpenStack.NodeCIDR,
			NetworkName:        config.OpenStack.NetworkName,
			SubnetName:         config.OpenStack.SubnetName,
			ExternalNetworkID:  config.OpenStack.ExternalNetworkID,
			Bastion: &openstack.Bastion{
				MachineConfig: openstack.MachineConfig(config.OpenStack.Bastion),
			},
			NodeClasses: []openstack.NodeClass{
				{
					Name:          "default",
					MachineConfig: openstack.MachineConfig(config.OpenStack.Worker),
				},
			},
			ControlPlane: &openstack.ControlPlane{
				MachineConfig:     openstack.MachineConfig(config.OpenStack.ControlPlane),
				Replicas:          controlPlaneReplicas,
				AvailabilityZones: config.ControlPlaneAZ,
			},
			NodePools: []openstack.NodePool{
				{
					Class:         "default",
					FailureDomain: config.OpenStack.WorkerFailureDomain,
					Name:          "default",
					Replicas:      config.OpenStack.WorkerReplicas,
				},
			},
		}

		if config.OIDC.IssuerURL != "" {
			flagValues.OIDC = &openstack.OIDC{
				IssuerURL:     config.OIDC.IssuerURL,
				CAFile:        config.OIDC.CAFile,
				ClientID:      config.OIDC.ClientID,
				UsernameClaim: config.OIDC.UsernameClaim,
				GroupsClaim:   config.OIDC.GroupsClaim,
			}
		}

		configData, err := openstack.GenerateClusterValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
			Name:      configMapName,
			Namespace: fmt.Sprintf("org-%s", config.Organization),
			Data:      configData,
		})
		if err != nil {
			return microerror.Mask(err)
		}

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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), "cluster-openstack", config.App.ClusterCatalog)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		clusterAppConfig := templateapp.Config{
			AppName:                 fmt.Sprintf("%s-cluster", config.Name),
			Catalog:                 config.App.ClusterCatalog,
			InCluster:               true,
			Name:                    "cluster-openstack",
			Namespace:               fmt.Sprintf("org-%s", config.Organization),
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

func templateDefaultAppsOpenstack(ctx context.Context, k8sClient k8sclient.Interface, output io.Writer, config ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := fmt.Sprintf("%s-userconfig", appName)

	var configMapYAML []byte
	{
		flagValues := openstack.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		configData, err := openstack.GenerateDefaultAppsValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(templateapp.UserConfig{
			Name:      configMapName,
			Namespace: fmt.Sprintf("org-%s", config.Organization),
			Data:      configData,
		})
		if err != nil {
			return microerror.Mask(err)
		}

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
			appVersion, err = getLatestVersion(ctx, k8sClient.CtrlClient(), "default-apps-openstack", config.App.DefaultAppsCatalog)
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
			Name:                    "default-apps-openstack",
			Namespace:               fmt.Sprintf("org-%s", config.Organization),
			Version:                 appVersion,
			UserConfigConfigMapName: configMapName,
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
