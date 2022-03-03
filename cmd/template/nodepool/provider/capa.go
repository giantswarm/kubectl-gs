package provider

import (
	"context"
	"io"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/nodepool/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPAEKSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	var err error

	data := struct {
		AvailabilityZones []string
		AWSInstanceType   string
		ClusterName       string
		Description       string
		MaxSize           int
		MinSize           int
		Name              string
		Namespace         string
		Organization      string
		Replicas          int
		ReleaseVersion    string
	}{
		AvailabilityZones: config.AvailabilityZones,
		AWSInstanceType:   config.AWSInstanceType,
		ClusterName:       config.ClusterName,
		Description:       config.Description,
		MaxSize:           config.NodesMax,
		MinSize:           config.NodesMin,
		Name:              config.NodePoolName,
		Namespace:         key.OrganizationNamespaceFromName(config.Organization),
		Organization:      config.Organization,
		Replicas:          config.NodesMin,
		ReleaseVersion:    config.ReleaseVersion,
	}

	err = runMutation(ctx, client, data, aws.GetEKSTemplates(), out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	var err error

	if config.UseAlikeInstanceTypes {
		return microerror.Maskf(invalidFlagError, "--use-alike-instance-types setting is not available for release %v", config.ReleaseVersion)
	}

	var sshSSOPublicKey string
	{
		sshSSOPublicKey, err = key.SSHSSOPublicKey(ctx, client.CtrlClient())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	data := struct {
		AvailabilityZones                   []string
		AWSInstanceType                     string
		ClusterName                         string
		Description                         string
		KubernetesVersion                   string
		MaxSize                             int
		MinSize                             int
		Name                                string
		Namespace                           string
		OnDemandBaseCapacity                int
		OnDemandPercentageAboveBaseCapacity int
		Organization                        string
		Replicas                            int
		SSHDConfig                          string
		SSOPublicKey                        string
		SudoConfig                          string
		ReleaseVersion                      string
	}{
		AvailabilityZones:                   config.AvailabilityZones,
		AWSInstanceType:                     config.AWSInstanceType,
		ClusterName:                         config.ClusterName,
		Description:                         config.Description,
		KubernetesVersion:                   "v1.19.9",
		MaxSize:                             config.NodesMax,
		MinSize:                             config.NodesMin,
		Name:                                config.NodePoolName,
		Namespace:                           key.OrganizationNamespaceFromName(config.Organization),
		OnDemandBaseCapacity:                config.OnDemandBaseCapacity,
		OnDemandPercentageAboveBaseCapacity: config.OnDemandPercentageAboveBaseCapacity,
		Organization:                        config.Organization,
		Replicas:                            config.NodesMin,
		SSHDConfig:                          key.NodeSSHDConfigEncoded(),
		SSOPublicKey:                        sshSSOPublicKey,
		SudoConfig:                          key.UbuntuSudoersConfigEncoded(),
		ReleaseVersion:                      config.ReleaseVersion,
	}

	err = runMutation(ctx, client, data, aws.GetAWSTemplates(), out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
