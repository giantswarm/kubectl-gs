package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v3/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Provider, options.Name, options.Namespace, options.ClusterName)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Provider, options.Namespace, options.ClusterName)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getByName(ctx context.Context, provider, name, namespace, clusterName string) (Resource, error) {
	var err error

	var np Resource
	{
		switch provider {
		case key.ProviderAWS:
			np, err = s.getByIdAWS(ctx, name, namespace, clusterName)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderAzure:
			np, err = s.getByIdAzure(ctx, name, namespace, clusterName)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderCAPA:
			np, err = s.getByIdCAPA(ctx, name, namespace, clusterName)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderCAPZ:
			np, err = s.getByIdCAPI(ctx, name, namespace, clusterName)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderCloudDirector:
			np, err = s.getByIdCAPI(ctx, name, namespace, clusterName)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		case key.ProviderVSphere:
			np, err = s.getByIdCAPI(ctx, name, namespace, clusterName)
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

		case key.ProviderCAPA:
			npCollection, err = s.getAllCAPA(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderCAPZ:
			npCollection, err = s.getAllCAPI(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderCloudDirector:
			npCollection, err = s.getAllCAPI(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderVSphere:
			npCollection, err = s.getAllCAPI(ctx, namespace, clusterID)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return npCollection, nil
}
