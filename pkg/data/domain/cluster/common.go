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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAll(ctx context.Context, namespace string) (Resource, error) {
	clusterList := &unstructured.UnstructuredList{}
	clusterList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "ClusterList",
	})

	err := s.client.List(ctx, clusterList, runtimeClient.InNamespace(namespace))
	if apierrors.IsForbidden(err) {
		return nil, microerror.Mask(insufficientPermissionsError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusterList.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	var clusterAppList application.AppList
	err = s.client.List(ctx, &clusterAppList, runtimeClient.InNamespace(namespace))
	if apierrors.IsForbidden(err) {
		return nil, microerror.Mask(insufficientPermissionsError)
	} else if err != nil {
		return nil, microerror.Mask(err)
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
	for i := range clusterList.Items {
		item := &clusterList.Items[i]
		cluster := Cluster{
			Cluster: item,
		}

		appKey := runtimeClient.ObjectKey{
			Namespace: item.GetNamespace(),
			Name:      item.GetName(),
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

func (s *Service) getByName(ctx context.Context, name, namespace string) (Resource, error) {
	var cluster Cluster

	capiCluster := &unstructured.Unstructured{}
	capiCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Cluster",
	})

	err := s.client.Get(ctx, runtimeClient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, capiCluster)
	if apierrors.IsForbidden(err) {
		return nil, microerror.Mask(insufficientPermissionsError)
	} else if apierrors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	cluster.Cluster = capiCluster

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
		resource, err = s.getByName(ctx, options.Name, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
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
