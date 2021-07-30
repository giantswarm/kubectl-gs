package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAzure(ctx context.Context, namespace string) (Resource, error) {
	var err error

	inNamespace := runtimeClient.InNamespace(namespace)

	var azureClusters map[string]*capzv1alpha3.AzureCluster
	{
		clusterCollection := &capzv1alpha3.AzureClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterCollection, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		azureClusters = make(map[string]*capzv1alpha3.AzureCluster, len(clusterCollection.Items))
		for _, cluster := range clusterCollection.Items {
			c := cluster
			azureClusters[cluster.GetName()] = &c
		}
	}

	clusters := &capiv1alpha3.ClusterList{}
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

			if azureCluster, exists := azureClusters[cr.GetName()]; exists {
				cr.TypeMeta = metav1.TypeMeta{
					APIVersion: "cluster.x-k8s.io/v1alpha3",
					Kind:       "Cluster",
				}
				azureCluster.TypeMeta = metav1.TypeMeta{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
					Kind:       "AzureCluster",
				}

				c := Cluster{
					Cluster:      &o,
					AzureCluster: azureCluster,
				}
				clusterCollection.Items = append(clusterCollection.Items, c)
			}
		}
	}

	return clusterCollection, nil
}

func (s *Service) getByNameAzure(ctx context.Context, name, namespace string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		capiv1alpha3.ClusterLabelName: name,
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	cluster := &Cluster{}

	{
		crs := &capiv1alpha3.ClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
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

	{
		crs := &capzv1alpha3.AzureClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		cluster.AzureCluster = &crs.Items[0]

		cluster.AzureCluster.TypeMeta = metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
			Kind:       "AzureCluster",
		}
	}

	return cluster, nil
}
