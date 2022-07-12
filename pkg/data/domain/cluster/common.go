package cluster

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Provider, options.Name, options.Namespace)
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

func (s *Service) getByName(ctx context.Context, provider, name, namespace string) (Resource, error) {
	var err error

	var cluster Resource
	{
		switch provider {
		case key.ProviderAWS:
			cluster, err = s.getByNameAWS(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderAzure:
			cluster, err = s.getByNameAzure(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderOpenStack:
			cluster, err = s.getByNameOpenStack(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderGCP:
			cluster, err = s.getByNameGCP(ctx, name, namespace)
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

		case key.ProviderOpenStack:
			clusterCollection, err = s.getAllOpenStack(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderGCP:
			clusterCollection, err = s.getAllGCP(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			return nil, microerror.Mask(invalidProviderError)
		}
	}

	return clusterCollection, nil
}

func (s *Service) Patch(ctx context.Context, object client.Object, options PatchOptions) error {
	var err error

	bytes, err := json.Marshal(options.PatchSpecs)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.client.Patch(ctx, object, client.RawPatch(types.JSONPatchType, bytes))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
