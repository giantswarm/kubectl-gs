package cluster

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
)

func (s *Service) V4ListAWS(ctx context.Context, options *ListOptions) (*providerv1alpha1.AWSConfigList, error) {
	clusters := &providerv1alpha1.AWSConfigList{}
	err := s.client.K8sClient.CtrlClient().List(ctx, clusters)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, nil
}

func (s *Service) V5ListAWS(ctx context.Context, options *ListOptions) (*infrastructurev1alpha2.AWSClusterList, error) {
	clusterIDs, err := s.getClusterIDs(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	awsClusters := &infrastructurev1alpha2.AWSClusterList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, awsClusters)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(awsClusters.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		var clusterList []infrastructurev1alpha2.AWSCluster
		for _, cluster := range awsClusters.Items {
			if _, exists := clusterIDs[cluster.Name]; exists {
				clusterList = append(clusterList, cluster)
			}
		}
		awsClusters.Items = clusterList
	}

	return awsClusters, nil
}
