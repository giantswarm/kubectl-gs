package provider

import (
	"context"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"io"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
)

func WriteOpenStackTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		ReleaseVersion    string
		SSHPublicKey      string

		Cloud                     string // OPENSTACK_CLOUD
		ControlPlaneMachineFlavor string // OPENSTACK_CONTROL_PLANE_MACHINE_FLAVOR
		DNSNameservers            string // OPENSTACK_DNS_NAMESERVERS
		FailureDomain             string // OPENSTACK_FAILURE_DOMAIN
		ImageName                 string // OPENSTACK_IMAGE_NAME
		NodeMachineFlavor         string // OPENSTACK_NODE_MACHINE_FLAVOR
		SSHKeyName                string // OPENSTACK_SSH_KEY_NAME
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.20.1",
		Name:              config.Name,
		Namespace:         config.Namespace,
		Organization:      config.Organization,
		ReleaseVersion:    config.ReleaseVersion,

		Cloud:                     config.Cloud,
		ControlPlaneMachineFlavor: config.ControlPlaneMachineFlavor,
		DNSNameservers:            config.DNSNameservers,
		FailureDomain:             config.FailureDomain,
		ImageName:                 config.ImageName,
		NodeMachineFlavor:         config.NodeMachineFlavor,
		SSHKeyName:                config.SSHKeyName,
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
