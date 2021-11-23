package provider

import (
	"context"
	"io"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
)

func WriteOpenStackTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		PodsCIDR          string
		ReleaseVersion    string
		SSHPublicKey      string

		Cloud                string   // OPENSTACK_CLOUD
		CloudConfig          string   // <no equivalent env var>
		DNSNameservers       []string // OPENSTACK_DNS_NAMESERVERS
		ExternalNetworkID    string   // <no equivalent env var>
		FailureDomain        string   // OPENSTACK_FAILURE_DOMAIN
		ImageName            string   // OPENSTACK_IMAGE_NAME
		NodeMachineFlavor    string   // OPENSTACK_NODE_MACHINE_FLAVOR
		RootVolumeDiskSize   string   // <no equivalent env var>
		RootVolumeSourceType string   // <no equivalent env var>
		RootVolumeSourceUUID string   // <no equivalent env var>
		SSHKeyName           string   // OPENSTACK_SSH_KEY_NAME
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.20.1",
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
		NodeMachineFlavor:    config.NodeMachineFlavor,
		RootVolumeDiskSize:   config.RootVolumeDiskSize,
		RootVolumeSourceType: config.RootVolumeSourceType,
		RootVolumeSourceUUID: config.RootVolumeSourceUUID,
		SSHKeyName:           config.SSHKeyName,
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
