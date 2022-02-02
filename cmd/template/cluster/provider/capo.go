package provider

import (
	"context"
	"fmt"
	"os"
	"text/template"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
)

func WriteOpenStackTemplate(ctx context.Context, k8sClient k8sclient.Interface, output *os.File, config ClusterCRsConfig) error {
	var userConfigConfigMapYaml []byte
	var err error

	clusterAppOutput := templateapp.AppCROutput{}
	defaultAppsAppOutput := templateapp.AppCROutput{}

	clusterVersion := config.ClusterVersion
	if clusterVersion == "" {
		var catalogEntryList applicationv1alpha1.AppCatalogEntryList
		err = k8sClient.CtrlClient().List(ctx, &catalogEntryList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app.kubernetes.io/name":            "cluster-openstack",
				"application.giantswarm.io/catalog": config.ClusterCatalog,
				"latest":                            "true",
			}),
			Namespace: "default",
		})

		if err != nil {
			return microerror.Mask(err)
		} else if len(catalogEntryList.Items) != 1 {
			return nil
		}

		clusterVersion = catalogEntryList.Items[0].Spec.Version
	}

	defaultAppsVersion := config.DefaultAppsVersion
	if defaultAppsVersion == "" {
		var catalogEntryList applicationv1alpha1.AppCatalogEntryList
		err = k8sClient.CtrlClient().List(ctx, &catalogEntryList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app.kubernetes.io/name":            "default-apps-openstack",
				"application.giantswarm.io/catalog": config.DefaultAppsCatalog,
				"latest":                            "true",
			}),
			Namespace: "default",
		})

		if err != nil {
			return microerror.Mask(err)
		} else if len(catalogEntryList.Items) != 1 {
			return nil
		}

		defaultAppsVersion = catalogEntryList.Items[0].Spec.Version
	}

	clusterAppConfig := templateapp.Config{
		AppName:   config.Name,
		Catalog:   config.ClusterCatalog,
		InCluster: true,
		Name:      "cluster-openstack",
		Namespace: fmt.Sprintf("org-%s", config.Organization),
		Version:   clusterVersion,
	}

	defaultAppsAppConfig := templateapp.Config{
		AppName:   fmt.Sprintf("%s-default-apps", config.Name),
		Catalog:   config.DefaultAppsCatalog,
		InCluster: true,
		Name:      "default-apps-openstack",
		Namespace: fmt.Sprintf("org-%s", config.Organization),
		Version:   defaultAppsVersion,
	}

	userConfig := templateapp.UserConfig{
		Namespace: fmt.Sprintf("org-%s", config.Organization),
	}

	{
		userConfig.Name = fmt.Sprintf("%s-userconfig", clusterAppConfig.AppName)

		flagValues := openstack.ClusterConfig{
			CloudName:         config.Cloud,
			CloudConfig:       config.CloudConfig,
			DNSNameservers:    config.DNSNameservers,
			ExternalNetworkID: config.ExternalNetworkID,
			NodeCIDR:          config.NodeCIDR,
			Organization:      config.Organization,
		}

		if config.EnableOIDC {
			flagValues.OIDC = &openstack.OIDC{
				Enabled: true,
			}
		}

		var fileValues openstack.ClusterConfig
		if config.BaseConfig != "" {
			content, err := os.ReadFile(config.BaseConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			err = yaml.Unmarshal(content, &fileValues)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		userConfig.Data, err = openstack.GenerateClusterConfigMapValues(flagValues, fileValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(userConfig)
		if err != nil {
			return microerror.Mask(err)
		}
		clusterAppConfig.UserConfigConfigMapName = userConfigMap.GetName()

		userConfigConfigMapYaml, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}

		clusterAppOutput.UserConfigConfigMap = string(userConfigConfigMapYaml)
	}

	{
		userConfig.Name = fmt.Sprintf("%s-userconfig", defaultAppsAppConfig.AppName)

		flagValues := openstack.DefaultAppsConfig{
			ClusterName:  config.Name,
			Organization: config.Organization,
		}

		if config.EnableOIDC {
			flagValues.OIDC = &openstack.OIDC{
				Enabled: true,
			}
		}

		userConfig.Data, err = openstack.GenerateDefaultAppsConfigMapValues(flagValues)
		if err != nil {
			return microerror.Mask(err)
		}

		userConfigMap, err := templateapp.NewConfigMap(userConfig)
		if err != nil {
			return microerror.Mask(err)
		}
		defaultAppsAppConfig.UserConfigConfigMapName = userConfigMap.GetName()

		userConfigConfigMapYaml, err = yaml.Marshal(userConfigMap)
		if err != nil {
			return microerror.Mask(err)
		}

		defaultAppsAppOutput.UserConfigConfigMap = string(userConfigConfigMapYaml)
	}

	clusterAppCRYaml, err := templateapp.NewAppCR(clusterAppConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterAppOutput.AppCR = string(clusterAppCRYaml)

	defaultAppsAppCRYaml, err := templateapp.NewAppCR(defaultAppsAppConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	defaultAppsAppOutput.AppCR = string(defaultAppsAppCRYaml)

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err = t.Execute(output, clusterAppOutput)
	if err != nil {
		return microerror.Mask(err)
	}

	err = t.Execute(output, defaultAppsAppOutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
