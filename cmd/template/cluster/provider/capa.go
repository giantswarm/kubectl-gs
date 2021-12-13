package provider

import (
	"context"
	"io"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var err error

	var sshSSOPublicKey string
	{
		sshSSOPublicKey, err = key.SSHSSOPublicKey(ctx, client.CtrlClient())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	data := struct {
		BastionSSHDConfig string
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		PodsCIDR          string
		ReleaseVersion    string
		SSHDConfig        string
		SSOPublicKey      string
	}{
		BastionSSHDConfig: key.BastionSSHDConfigEncoded(),
		Description:       config.Description,
		KubernetesVersion: "v1.19.9",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Organization),
		Organization:      config.Organization,
		PodsCIDR:          config.PodsCIDR,
		ReleaseVersion:    config.ReleaseVersion,
		SSHDConfig:        key.NodeSSHDConfigEncoded(),
		SSOPublicKey:      sshSSOPublicKey,
	}

	var templates []templateConfig
	for _, t := range aws.GetAWSTemplates() {
		templates = append(templates, templateConfig(t))
	}

	err = runMutation(ctx, data, templates, out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteCAPAEKSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var err error

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		ReleaseVersion    string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.21",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Organization),
		Organization:      config.Organization,
		ReleaseVersion:    config.ReleaseVersion,
	}

	var templates []templateConfig
	for _, t := range aws.GetEKSTemplates() {
		templates = append(templates, templateConfig(t))
	}

	err = runMutation(ctx, data, templates, out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
