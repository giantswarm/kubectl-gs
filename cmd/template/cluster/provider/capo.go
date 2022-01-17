package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	templateapp "github.com/giantswarm/kubectl-gs/cmd/template/app"
	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/microerror"
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

func WriteOpenStackTemplateAppCR(ctx context.Context, runner *templateapp.Runner, config ClusterCRsConfig) error {
	var err error

	clusterAppFlags := &templateapp.Flag{
		AppName:           config.Name,
		Catalog:           "control-plane-catalog",
		InCluster:         true,
		Name:              "cluster-openstack",
		Namespace:         fmt.Sprintf("org-%s", config.Organization),
		Version:           config.ClusterAppVersion,
		FlagUserConfigMap: config.ClusterUserConfigMap,
	}

	runner.Flag = clusterAppFlags
	err = runner.Run(nil, []string{})
	if err != nil {
		return microerror.Mask(err)
	}

	defaultAppsAppFlags := &templateapp.Flag{
		AppName:           "openstack-default-apps",
		Catalog:           "control-plane-catalog",
		Cluster:           config.Name,
		Name:              "default-apps-openstack",
		Namespace:         fmt.Sprintf("org-%s", config.Organization),
		Version:           config.DefaultAppsAppVersion,
		FlagUserConfigMap: config.DefaultAppsUserConfigMap,
	}

	runner.Flag = defaultAppsAppFlags
	err = runner.Run(nil, []string{})
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
