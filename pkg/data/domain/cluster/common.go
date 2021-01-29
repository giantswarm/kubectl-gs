package cluster

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.ID) > 0 {
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

func (s *Service) getById(ctx context.Context, provider, id, namespace string) (Resource, error) {
	var err error

	var cluster Resource
	{
		switch provider {
		case key.ProviderAWS:
			cluster, err = s.getByIdAWS(ctx, id, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			cluster, err = s.getByIdAzure(ctx, id, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return cluster, nil
}

func (s *Service) getAll(ctx context.Context, provider, namespace string) (Resource, error) {
	var err error

	var clusterCollection Resource
	{
		switch provider {
		case key.ProviderAWS:
			clusterCollection, err = s.getAllAWS(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			clusterCollection, err = s.getAllAzure(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return clusterCollection, nil
}
