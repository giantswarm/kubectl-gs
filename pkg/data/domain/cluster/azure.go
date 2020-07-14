package cluster

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
)

func (s *Service) V4ListAzure(ctx context.Context, options *ListOptions) (*providerv1alpha1.AzureConfigList, error) {
	clusters := &providerv1alpha1.AzureConfigList{}
	err := s.client.K8sClient.CtrlClient().List(ctx, clusters)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, nil
}
