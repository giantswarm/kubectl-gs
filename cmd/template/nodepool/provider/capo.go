package provider

import (
	"context"
	"io"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/nodepool/provider/templates/openstack"
)

func WriteCAPOTemplate(ctx context.Context, out io.Writer, config NodePoolCRsConfig) error {
	var err error

	data := struct {
		Description       string
		KubernetesVersion string
		ClusterName       string
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
		ClusterName:       config.ClusterName,
		Name:              config.NodePoolID,
		Namespace:         config.Namespace,
		Organization:      config.Organization,
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

	err = runMutation(ctx, nil, data, openstack.GetTemplates(), out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
