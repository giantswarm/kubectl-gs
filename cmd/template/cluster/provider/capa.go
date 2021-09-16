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

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Owner             string
		ReleaseVersion    string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.19.9",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Owner),
		Owner:             config.Owner,
		ReleaseVersion:    config.ReleaseVersion,
	}

	err = runMutation(ctx, client, data, aws.GetTemplates(), out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
