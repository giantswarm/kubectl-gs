package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/kubectl-gs/internal/key"
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

func (s *Service) ListForProvider(ctx context.Context, provider string) (*CommonClusterList, error) {
	var err error

	var clusters []runtime.Object
	{
		listOptions := &ListOptions{}

		switch provider {
		case key.ProviderAWS:
			clusters, err = s.GetAllAWSLists(ctx, listOptions)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			clusters, err = s.GetAllAzureLists(ctx, listOptions)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderKVM:
			clusters, err = s.GetAllKVMLists(ctx, listOptions)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}
	}

	clusterCR := &CommonClusterList{}
	{
		clusterCR.APIVersion = "v1"
		clusterCR.Kind = "List"
		clusterCR.Items = clusters
	}

	return clusterCR, nil
}
