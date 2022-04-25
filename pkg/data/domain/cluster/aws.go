package cluster

import (
	"context"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) (Resource, error) {
	var err error

	inNamespace := runtimeClient.InNamespace(namespace)

	var awsClusters map[string]*infrastructurev1alpha3.AWSCluster
	{
		clusterCollection := &infrastructurev1alpha3.AWSClusterList{}
		err = s.client.List(ctx, clusterCollection, inNamespace)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		awsClusters = make(map[string]*infrastructurev1alpha3.AWSCluster, len(clusterCollection.Items))
		for _, cluster := range clusterCollection.Items {
			c := cluster
			awsClusters[cluster.GetName()] = &c
		}
	}

	clusters := &capiv1beta1.ClusterList{}
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

			if awsCluster, exists := awsClusters[cr.GetName()]; exists {
				cr.TypeMeta = metav1.TypeMeta{
					APIVersion: "cluster.x-k8s.io/v1beta1",
					Kind:       "Cluster",
				}
				awsCluster.TypeMeta = infrastructurev1alpha3.NewAWSClusterTypeMeta()

				c := Cluster{
					Cluster:    &o,
					AWSCluster: awsCluster,
				}
				clusterCollection.Items = append(clusterCollection.Items, c)
			}
		}
	}

	return clusterCollection, nil
}

func (s *Service) getByNameAWS(ctx context.Context, name, namespace string) (Resource, error) {
	var err error

	inNamespace := runtimeClient.InNamespace(namespace)

	cluster := &Cluster{}

	{
		labelSelector := runtimeClient.MatchingLabels{
			capiv1beta1.ClusterLabelName: name,
		}
		crs := &capiv1beta1.ClusterList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			// Fall back on old giant swarm cluster id label
			labelSelector = runtimeClient.MatchingLabels{
				label.Cluster: name,
			}
			err = s.client.List(ctx, crs, labelSelector, inNamespace)
			if apierrors.IsForbidden(err) {
				return nil, microerror.Mask(insufficientPermissionsError)
			} else if err != nil {
				return nil, microerror.Mask(err)
			}

			if len(crs.Items) < 1 {
				return nil, microerror.Mask(notFoundError)
			}
		}
		cluster.Cluster = &crs.Items[0]

		cluster.Cluster.TypeMeta = metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "Cluster",
		}
	}

	{
		labelSelector := runtimeClient.MatchingLabels{
			label.Cluster: name,
		}
		crs := &infrastructurev1alpha3.AWSClusterList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if apierrors.IsForbidden(err) {
			return nil, microerror.Mask(insufficientPermissionsError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		cluster.AWSCluster = &crs.Items[0]

		cluster.AWSCluster.TypeMeta = infrastructurev1alpha3.NewAWSClusterTypeMeta()
	}

	return cluster, nil
}
