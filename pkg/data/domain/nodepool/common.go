package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (NodepoolCollection, error) {
	var npCollection NodepoolCollection

	if len(options.ID) > 0 {
		np, err := s.getById(ctx, options.Provider, options.ID, options.Namespace)
		if err != nil {
			return NodepoolCollection{}, microerror.Mask(err)
		}

		npCollection.Items = append(npCollection.Items, np)
	} else {
		npList, err := s.getAll(ctx, options.Provider, options.Namespace)
		if err != nil {
			return NodepoolCollection{}, microerror.Mask(err)
		}

		npCollection.Items = npList
	}

	return npCollection, nil
}

func (s *Service) getById(ctx context.Context, provider, id, namespace string) (Nodepool, error) {
	var err error

	var np Nodepool
	{
		switch provider {
		case key.ProviderAWS:
			np, err = s.getByIdAWS(ctx, id, namespace)
			if err != nil {
				return Nodepool{}, microerror.Mask(err)
			}

		case key.ProviderAzure:
			np, err = s.getByIdAzure(ctx, id, namespace)
			if err != nil {
				return Nodepool{}, microerror.Mask(err)
			}

		default:
			return Nodepool{}, microerror.Mask(invalidProviderError)
		}
	}

	return np, nil
}

func (s *Service) getAll(ctx context.Context, provider, namespace string) ([]Nodepool, error) {
	var err error

	var npCollection []Nodepool
	{
		switch provider {
		case key.ProviderAWS:
			npCollection, err = s.getAllAWS(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			npCollection, err = s.getAllAzure(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return npCollection, nil
}
