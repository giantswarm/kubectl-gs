package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options *GetOptions) (runtime.Object, error) {
	var err error

	var resource runtime.Object
	if options.ID != "" {
		resource, err = s.getById(ctx, options.Provider, options.ID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Provider)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getById(ctx context.Context, provider string, id string) (runtime.Object, error) {
	var err error

	var resource runtime.Object
	{
		switch provider {
		case key.ProviderAWS:
			resource, err = s.getByIdAWS(ctx, id)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			resource, err = s.getByIdAzure(ctx, id)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderKVM:
			resource, err = s.getByIdKVM(ctx, id)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return resource, nil
}

func (s *Service) getAll(ctx context.Context, provider string) (runtime.Object, error) {
	var err error

	var clusters []runtime.Object
	{
		switch provider {
		case key.ProviderAWS:
			clusters, err = s.getAllAWS(ctx)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			clusters, err = s.getAllAzure(ctx)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderKVM:
			clusters, err = s.getAllKVM(ctx)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	resource := &CommonClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: clusters,
	}

	return resource, nil
}

func (s *Service) getClusterIDs(ctx context.Context) (map[string]bool, error) {
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
