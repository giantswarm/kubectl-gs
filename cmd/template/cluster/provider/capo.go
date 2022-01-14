package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/pkg/template/app"
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

func WriteOpenStackTemplateAppCR(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var userConfigConfigMapYaml []byte
	var err error

	appConfig := app.Config{
		AppName:                 config.Name,
		Catalog:                 "control-plane-catalog",
		UserConfigConfigMapName: fmt.Sprintf("%s-%s", config.Name, "userconfig"),
		Cluster:                 config.Namespace,
		InCluster:               true,
		Name:                    "cluster-openstack",
		Namespace:               config.Namespace,
		Version:                 "0.1.0",
	}

	type RootVolume struct {
		Enabled    bool   `json:"enabled,omitempty"`
		SourceUUID string `json:"sourceUUID,omitempty"`
	}

	rootVol := &RootVolume{
		Enabled:    config.RootVolumeSourceUUID != "",
		SourceUUID: config.RootVolumeSourceUUID,
	}

	{
		rv, err := yaml.Marshal(rootVol)
		if err != nil {
			return microerror.Mask(err)
		} else if len(rv) == 3 {
			rootVol = nil
		}
	}

	values := struct {
		Description       string      `json:"clusterDescription,omitempty"`
		KubernetesVersion string      `json:"kubernetesVersion,omitempty"`
		Organization      string      `json:"organization"`
		ReleaseVersion    string      `json:"releaseVersion"`
		ExternalNetworkID string      `json:"externalNetworkID,omitempty"`
		Cloud             string      `json:"cloudName,omitempty"`
		CloudConfig       string      `json:"cloudConfig,omitempty"`
		RootVolume        *RootVolume `json:"rootVolume,omitempty"`
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.20.9",
		Organization:      config.Organization,
		Cloud:             config.Cloud,
		ReleaseVersion:    config.ReleaseVersion,
		CloudConfig:       config.CloudConfig,
		ExternalNetworkID: config.ExternalNetworkID,
		RootVolume:        rootVol,
	}

	data, err := yaml.Marshal(values)

	configMapConfig := app.ConfigMapConfig{
		Data:      string(data),
		Name:      appConfig.UserConfigConfigMapName,
		Namespace: appConfig.Namespace,
	}

	userConfigMap, err := app.NewConfigMap(configMapConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	userConfigConfigMapYaml, err = yaml.Marshal(userConfigMap)
	if err != nil {
		return microerror.Mask(err)
	}

	appCRYaml, err := app.NewAppCR(appConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	type AppCROutput struct {
		AppCR               string
		UserConfigConfigMap string
		UserConfigSecret    string
	}

	appCROutput := AppCROutput{
		AppCR:               string(appCRYaml),
		UserConfigSecret:    string([]byte{}),
		UserConfigConfigMap: string(userConfigConfigMapYaml),
	}

	t := template.Must(template.New("appCR").Parse(key.AppCRTemplate))

	err = t.Execute(os.Stdout, appCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
