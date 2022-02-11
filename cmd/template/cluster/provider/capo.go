package provider

import (
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
)

func WriteOpenStackTemplate(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterConfig) error {
	err := templateClusterOpenstack(ctx, k8sClient, output, config)
	if err != nil {
		return microerror.Mask(err)
	}

	err = templateDefaultAppsOpenstack(ctx, k8sClient, output, config)
	return microerror.Mask(err)
}

func templateClusterOpenstack(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterConfig) error {
	appName := config.Name
	configMapName := fmt.Sprintf("%s-userconfig", appName)

	var configMapYAML []byte
	{
		flagValues := openstack.ClusterConfig{
			ClusterDescription: config.Description,
			DNSNameservers:     config.OpenStack.DNSNameservers,
			Organization:       config.Organization,
			CloudConfig:        config.OpenStack.CloudConfig,
			CloudName:          config.OpenStack.Cloud,
			NodeCIDR:           config.OpenStack.NodeCIDR,
			ExternalNetworkID:  config.OpenStack.ExternalNetworkID,
			Bastion: &openstack.Bastion{
				Flavor: config.OpenStack.BastionMachineFlavor,
				RootVolume: openstack.MachineRootVolume{
					DiskSize:   config.OpenStack.BastionDiskSize,
					SourceUUID: config.OpenStack.BastionImageUUID,
				},
			},
			RootVolume: &openstack.RootVolume{
				Enabled:    true,
				SourceUUID: config.OpenStack.NodeImageUUID,
			},
			NodeClasses: []openstack.NodeClass{
				{
					Name:          "default",
					MachineFlavor: config.OpenStack.WorkerMachineFlavor,
					DiskSize:      config.OpenStack.WorkerDiskSize,
				},
			},
			ControlPlane: &openstack.ControlPlane{
				MachineFlavor: config.OpenStack.ControlPlaneMachineFlavor,
				DiskSize:      config.OpenStack.ControlPlaneDiskSize,
				Replicas:      config.ControlPlaneReplicas,
			},
			NodePools: []openstack.NodePool{
				{
					Name:     "default",
					Class:    "default",
					Replicas: config.OpenStack.WorkerReplicas,
				},
			},
			OIDC: &openstack.OIDC{
				Enabled: config.OpenStack.EnableOIDC,
			},
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
			AppName:                 config.Name,
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

func templateDefaultAppsOpenstack(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterConfig) error {
	appName := fmt.Sprintf("%s-default-apps", config.Name)
	configMapName := fmt.Sprintf("%s-userconfig", appName)

	var configMapYAML []byte
	{
		flagValues := openstack.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
			OIDC: &openstack.OIDC{
				Enabled: config.OpenStack.EnableOIDC,
			},
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
