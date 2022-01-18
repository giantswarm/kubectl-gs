package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/internal/key"
	templateapp "github.com/giantswarm/kubectl-gs/pkg/template/app"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

func WriteOpenStackTemplateRaw(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		PodsCIDR          string
		ReleaseVersion    string

		Cloud                string   // OPENSTACK_CLOUD
		CloudConfig          string   // <no equivalent env var>
		DNSNameservers       []string // OPENSTACK_DNS_NAMESERVERS
		ExternalNetworkID    string   // <no equivalent env var>
		FailureDomain        string   // OPENSTACK_FAILURE_DOMAIN
		ImageName            string   // OPENSTACK_IMAGE_NAME
		NodeCIDR             string   // <no equivalent env var>
		NodeMachineFlavor    string   // OPENSTACK_NODE_MACHINE_FLAVOR
		RootVolumeDiskSize   string   // <no equivalent env var>
		RootVolumeSourceType string   // <no equivalent env var>
		RootVolumeSourceUUID string   // <no equivalent env var>
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.20.9",
		Name:              config.Name,
		Namespace:         config.Namespace,
		Organization:      config.Organization,
		PodsCIDR:          config.PodsCIDR,
		ReleaseVersion:    config.ReleaseVersion,

		Cloud:                config.Cloud,
		CloudConfig:          config.CloudConfig,
		DNSNameservers:       config.DNSNameservers,
		ExternalNetworkID:    config.ExternalNetworkID,
		FailureDomain:        config.FailureDomain,
		ImageName:            config.ImageName,
		NodeCIDR:             config.NodeCIDR,
		NodeMachineFlavor:    config.NodeMachineFlavor,
		RootVolumeDiskSize:   config.RootVolumeDiskSize,
		RootVolumeSourceType: config.RootVolumeSourceType,
		RootVolumeSourceUUID: config.RootVolumeSourceUUID,
	}

	var templates []templateConfig
	for _, t := range openstack.GetTemplates() {
		templates = append(templates, templateConfig(t))
	}

	err := runMutation(ctx, client, data, templates, out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteOpenStackTemplateAppCR(ctx context.Context, config ClusterCRsConfig) error {
	var userConfigConfigMapYaml []byte
	var err error

	clusterAppOutput := templateapp.AppCROutput{}
	defaultAppsAppOutput := templateapp.AppCROutput{}

	clusterAppConfig := templateapp.Config{
		AppName:   fmt.Sprintf("%s-cluster", config.Name),
		Catalog:   "control-plane-catalog",
		InCluster: true,
		Name:      "cluster-openstack",
		Namespace: fmt.Sprintf("org-%s", config.Organization),
		Version:   config.ClusterAppVersion,
	}

	defaultAppsAppConfig := templateapp.Config{
		AppName:   fmt.Sprintf("%s-default-apps", config.Name),
		Catalog:   "default",
		InCluster: true,
		Name:      "default-apps-openstack",
		Namespace: fmt.Sprintf("org-%s", config.Organization),
		Version:   config.DefaultAppsAppVersion,
	}

	userConfig := templateapp.UserConfig{
		Namespace: fmt.Sprintf("org-%s", config.Organization),
	}

	if config.ClusterAppUserConfigMap != "" {
		userConfig.Name = fmt.Sprintf("%s-userconfig", clusterAppConfig.AppName)
		userConfig.Path = config.ClusterAppUserConfigMap

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

	if config.DefaultAppsAppUserConfigMap != "" {
		userConfig.Name = fmt.Sprintf("%s-userconfig", defaultAppsAppConfig.AppName)
		userConfig.Path = config.DefaultAppsAppUserConfigMap

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

	err = t.Execute(os.Stdout, clusterAppOutput)
	if err != nil {
		return microerror.Mask(err)
	}

	err = t.Execute(os.Stdout, defaultAppsAppOutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
