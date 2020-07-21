package cluster

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) (*infrastructurev1alpha2.AWSClusterList, error) {
	var err error

	var clusterIDs map[string]bool
	{
		apiClusters := &apiv1alpha2.ClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, apiClusters)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(apiClusters.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		clusterIDs = make(map[string]bool, len(apiClusters.Items))
		for _, cluster := range apiClusters.Items {
			clusterIDs[cluster.Name] = true
		}
	}

	clusterList := &infrastructurev1alpha2.AWSClusterList{}
	options := &runtimeClient.ListOptions{
		Namespace: namespace,
	}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterList, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterList.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		var clusters []infrastructurev1alpha2.AWSCluster
		for _, cluster := range clusterList.Items {
			if _, exists := clusterIDs[cluster.Name]; exists {
				cluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()
				clusters = append(clusters, cluster)
			}
		}
		clusterList.Items = clusters
	}

	{
		clusterList.APIVersion = "v1"
		clusterList.Kind = "List"
	}

	return clusterList, nil
}

func (s *Service) getByIdAWS(ctx context.Context, id, namespace string) (*infrastructurev1alpha2.AWSCluster, error) {

	cluster := &infrastructurev1alpha2.AWSCluster{}
	key := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}
	err := s.client.K8sClient.CtrlClient().Get(ctx, key, cluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	cluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()

	return cluster, nil
}
