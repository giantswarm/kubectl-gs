package cluster

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v4/internal/key"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Provider, options.Name, options.Namespace, options.FallbackToCapi)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Provider, options.Namespace, options.FallbackToCapi)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getByName(ctx context.Context, provider, name, namespace string, fallbackToCapi bool) (Resource, error) {
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

		case key.ProviderGCP:
			cluster, err = s.getByNameGCP(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderOpenStack, key.ProviderCAPA, key.ProviderKVM, key.ProviderVSphere, key.ProviderCloudDirector:
			cluster, err = s.getByNameCommonCapi(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			if !fallbackToCapi {
				return nil, microerror.Mask(invalidProviderError)
			}

			cluster, err = s.getByNameCommonCapi(ctx, name, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}
	}

	return cluster, nil
}

func (s *Service) getAll(ctx context.Context, provider, namespace string, fallbackToCapi bool) (Resource, error) {
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

		case key.ProviderGCP:
			clusterCollection, err = s.getAllGCP(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		case key.ProviderOpenStack, key.ProviderCAPA, key.ProviderKVM, key.ProviderVSphere, key.ProviderCloudDirector:
			clusterCollection, err = s.getAllCommonCapi(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}

		default:
			if !fallbackToCapi {
				return nil, microerror.Mask(invalidProviderError)
			}

			clusterCollection, err = s.getAllCommonCapi(ctx, namespace)
			if err != nil {
				return nil, microerror.Mask(err)
			}
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
