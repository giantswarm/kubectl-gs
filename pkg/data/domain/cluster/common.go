package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (runtime.Object, error) {
	var err error

	var resource runtime.Object
	if options.ID != "" {
		resource, err = s.getById(ctx, options.Provider, options.ID, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Provider, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getById(ctx context.Context, provider, id, namespace string) (runtime.Object, error) {
	var err error

	var resource runtime.Object
	{
		switch provider {
		case key.ProviderAWS:
			resource, err = s.getByIdAWS(ctx, id, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			resource, err = s.getByIdAzure(ctx, id, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return resource, nil
}

func (s *Service) getAll(ctx context.Context, provider, namespace string) (runtime.Object, error) {
	var err error

	var clusterList runtime.Object
	{
		switch provider {
		case key.ProviderAWS:
			clusterList, err = s.getAllAWS(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			clusterList, err = s.getAllAzure(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return clusterList, nil
}
