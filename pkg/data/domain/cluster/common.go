package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

func (s *Service) getClusterIDs(ctx context.Context, options *ListOptions) (map[string]bool, error) {
	var clusterIDs map[string]bool

	apiClusters := &apiv1alpha2.ClusterList{}
	err := s.client.K8sClient.CtrlClient().List(ctx, apiClusters)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(apiClusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	clusterIDs = make(map[string]bool, len(apiClusters.Items))
	for _, cluster := range apiClusters.Items {
		clusterIDs[cluster.Name] = true
	}

	return clusterIDs, nil
}
