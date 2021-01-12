package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.ID) > 0 {
		resource, err = s.getById(ctx, options.Provider, options.ID, options.Namespace, options.ClusterID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Provider, options.Namespace, options.ClusterID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getById(ctx context.Context, provider, id, namespace, clusterID string) (Resource, error) {
	var err error

	var np Resource
	{
		switch provider {
		case key.ProviderAWS:
			np, err = s.getByIdAWS(ctx, id, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderAzure:
			np, err = s.getByIdAzure(ctx, id, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		default:
			return nil, microerror.Mask(invalidProviderError)
		}

	}

	return np, nil
}

func (s *Service) getAll(ctx context.Context, provider, namespace, clusterID string) (Resource, error) {
	var err error

	var npCollection Resource
	{
		switch provider {
		case key.ProviderAWS:
			npCollection, err = s.getAllAWS(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			npCollection, err = s.getAllAzure(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return npCollection, nil
}
