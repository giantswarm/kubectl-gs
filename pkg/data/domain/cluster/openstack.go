package cluster

import (
	"context"
	"fmt"

	application "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
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
			Name:      app.ObjectMeta.Labels[label.Cluster],
		}
		app.TypeMeta = meta.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "App",
		}
		switch app.ObjectMeta.Labels[label.AppKubernetesName] {
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
			APIVersion: "cluster.x-k8s.io/v1beta1",
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
			APIVersion: "cluster.x-k8s.io/v1beta1",
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
