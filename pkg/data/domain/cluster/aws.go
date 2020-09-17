package cluster

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v2/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) (*infrastructurev1alpha2.AWSClusterList, error) {
	var err error

	options := &runtimeClient.ListOptions{
		Namespace: namespace,
	}

	var clusterIDs map[string]bool
	{
		// The CAPI clusters have the labels. They can be used for filtering.
		apiClusters := &capiv1alpha2.ClusterList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, apiClusters, options)
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
	var err error

	objKey := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}

	// The CAPI cluster has the labels. It can be used for filtering.
	cluster := &capiv1alpha2.Cluster{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, cluster)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	awsCluster := &infrastructurev1alpha2.AWSCluster{}
	err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, awsCluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	awsCluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()

	return awsCluster, nil
}
