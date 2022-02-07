package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllOpenStack(ctx context.Context, namespace string) (Resource, error) {
	var err error

	inNamespace := runtimeClient.InNamespace(namespace)

	clusters := &capiv1alpha3.ClusterList{}
	{
		err = s.client.List(ctx, clusters, inNamespace)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusters.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	clusterCollection := &Collection{}
	{
		for _, cr := range clusters.Items {
			o := cr

			cr.TypeMeta = metav1.TypeMeta{
				APIVersion: "cluster.x-k8s.io/v1alpha3",
				Kind:       "Cluster",
			}
			c := Cluster{
				Cluster: &o,
			}
			clusterCollection.Items = append(clusterCollection.Items, c)
		}
	}

	return clusterCollection, nil
}

func (s *Service) getByNameOpenStack(ctx context.Context, name, namespace string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		capiv1alpha3.ClusterLabelName: name,
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	cluster := &Cluster{}

	{
		crs := &capiv1alpha3.ClusterList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		cluster.Cluster = &crs.Items[0]

		cluster.Cluster.TypeMeta = metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha3",
			Kind:       "Cluster",
		}
	}

	return cluster, nil
}
