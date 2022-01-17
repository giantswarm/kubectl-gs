package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	templateapp "github.com/giantswarm/kubectl-gs/cmd/template/app"
	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
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

func WriteOpenStackTemplateAppCR(ctx context.Context, cmdConfig templateapp.Config, config ClusterCRsConfig) error {
	var err error

	var clusterAppCmd *cobra.Command
	{
		clusterAppCmd, err = templateapp.New(cmdConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var defaultAppsAppCmd *cobra.Command
	{
		defaultAppsAppCmd, err = templateapp.New(cmdConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	clusterAppArgs := []string{
		fmt.Sprintf("--app-name=%s", config.Name),
		fmt.Sprintf("--namespace=org-%s", config.Organization),
		"--name=cluster-openstack",
		"--catalog=control-plane-catalog",
		"--in-cluster=true",
		"--version=0.1.0",
	}

	if config.UserConfigMap != "" {
		clusterAppArgs = append(clusterAppArgs, fmt.Sprintf("--user-configmap=%s", config.UserConfigMap))
	}

	// Need to replace the args before executing `template app` command, so that
	// it gets the right set and do not complain on extra ones.
	clusterAppCmd.SetArgs(clusterAppArgs)
	err = clusterAppCmd.Execute()
	if err != nil {
		return microerror.Mask(err)
	}

	defaultAppsAppArgs := []string{
		fmt.Sprintf("--cluster=%s", config.Name),
		"--app-name=openstack-default-apps",
		"--namespace=kube-system",
		"--name=default-apps-openstack",
		"--catalog=control-plane-catalog",
		"--version=0.1.0",
	}

	defaultAppsAppCmd.SetArgs(defaultAppsAppArgs)
	err = defaultAppsAppCmd.Execute()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
