package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	application "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capi "sigs.k8s.io/cluster-api/api/core/v1beta2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v5/internal/key"
)

func (s *Service) getAllCommonCapi(ctx context.Context, namespace string) (Resource, error) {
	var clusterList capi.ClusterList
	{
		err := s.client.List(ctx, &clusterList, runtimeClient.InNamespace(namespace))
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterList.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	var clusterAppList application.AppList
	{
		err := s.client.List(ctx, &clusterAppList, runtimeClient.InNamespace(namespace))
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterList.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	clusterAppMap := map[runtimeClient.ObjectKey]application.App{}
	defaultAppsAppMap := map[runtimeClient.ObjectKey]application.App{}
	for _, app := range clusterAppList.Items {
		clusterKey := runtimeClient.ObjectKey{
			Namespace: app.Namespace,
			Name:      app.Labels[label.Cluster],
		}
		app.TypeMeta = meta.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "App",
		}
		switch app.Labels[label.AppKubernetesName] {
		case "default-apps-openstack":
			defaultAppsAppMap[clusterKey] = app
		case "cluster-openstack":
			clusterAppMap[clusterKey] = app
		}
	}

	var clusterCollection Collection
	for _, capiCluster := range clusterList.Items {
		clusterCopy := capiCluster.DeepCopy()
		clusterCopy.TypeMeta = meta.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta2",
			Kind:       "Cluster",
		}
		cluster := Cluster{
			Cluster: clusterCopy,
		}

		appKey := runtimeClient.ObjectKey{
			Namespace: clusterCopy.Namespace,
			Name:      clusterCopy.Name,
		}
		if clusterApp, ok := clusterAppMap[appKey]; ok {
			cluster.ClusterApp = &clusterApp
		}
		if defaultAppsApp, ok := defaultAppsAppMap[appKey]; ok {
			cluster.DefaultAppsApp = &defaultAppsApp
		}

		clusterCollection.Items = append(clusterCollection.Items, cluster)
	}

	return &clusterCollection, nil
}

func (s *Service) getByNameCommonCapi(ctx context.Context, name, namespace string) (Resource, error) {
	var cluster Cluster

	{
		var capiCluster capi.Cluster
		err := s.client.Get(ctx, runtimeClient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &capiCluster)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		capiCluster.TypeMeta = meta.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta2",
			Kind:       "Cluster",
		}
		cluster.Cluster = &capiCluster
	}

	{
		var clusterApp application.App
		err := s.client.Get(ctx, runtimeClient.ObjectKey{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-cluster", name),
		}, &clusterApp)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			clusterApp.TypeMeta = meta.TypeMeta{
				APIVersion: "application.giantswarm.io/v1alpha1",
				Kind:       "App",
			}
			cluster.ClusterApp = &clusterApp
		}
	}

	{
		var defaultAppsApp application.App
		err := s.client.Get(ctx, runtimeClient.ObjectKey{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-default-apps", name),
		}, &defaultAppsApp)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			defaultAppsApp.TypeMeta = meta.TypeMeta{
				APIVersion: "application.giantswarm.io/v1alpha1",
				Kind:       "App",
			}
			cluster.DefaultAppsApp = &defaultAppsApp
		}
	}

	return &cluster, nil
}

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

func (s *Service) Patch(ctx context.Context, object runtimeClient.Object, options PatchOptions) error {
	var err error

	bytes, err := json.Marshal(options.PatchSpecs)
	if err != nil {
		return microerror.Mask(err)
	}

	err = s.client.Patch(ctx, object, runtimeClient.RawPatch(types.JSONPatchType, bytes))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
