package cluster

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/internal/label"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) (Resource, error) {
	var err error

	inNamespace := runtimeClient.InNamespace(namespace)

	var awsClusters map[string]*infrastructurev1alpha2.AWSCluster
	{
		clusterCollection := &infrastructurev1alpha2.AWSClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterCollection, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		awsClusters = make(map[string]*infrastructurev1alpha2.AWSCluster, len(clusterCollection.Items))
		for _, cluster := range clusterCollection.Items {
			c := cluster
			awsClusters[cluster.GetName()] = &c
		}
	}

	clusters := &capiv1alpha2.ClusterList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, clusters, inNamespace)
		if err != nil {
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
					APIVersion: "cluster.x-k8s.io/v1alpha2",
					Kind:       "Cluster",
				}
				awsCluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()

				c := Cluster{
					V1Alpha2Cluster: &o,
					AWSCluster:      awsCluster,
				}
				clusterCollection.Items = append(clusterCollection.Items, c)
			}
		}
	}

	return clusterCollection, nil
}

func (s *Service) getByNameAWS(ctx context.Context, name, namespace string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		label.Cluster: name,
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	cluster := &Cluster{}

	{
		crs := &capiv1alpha2.ClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		cluster.V1Alpha2Cluster = &crs.Items[0]

		cluster.V1Alpha2Cluster.TypeMeta = metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha2",
			Kind:       "Cluster",
		}
	}

	{
		crs := &infrastructurev1alpha2.AWSClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		cluster.AWSCluster = &crs.Items[0]

		cluster.AWSCluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()
	}

	return cluster, nil
}
