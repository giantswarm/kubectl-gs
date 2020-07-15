package cluster

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
)

func (s *Service) V4ListAWS(ctx context.Context, options *ListOptions) (*corev1alpha1.AWSClusterConfigList, error) {
	clusters := &corev1alpha1.AWSClusterConfigList{}
	err := s.client.K8sClient.CtrlClient().List(ctx, clusters)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, nil
}

func (s *Service) V5ListAWS(ctx context.Context, options *ListOptions) (*infrastructurev1alpha2.AWSClusterList, error) {
	clusterIDs, err := s.getClusterIDs(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	awsClusters := &infrastructurev1alpha2.AWSClusterList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, awsClusters)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(awsClusters.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		var clusterList []infrastructurev1alpha2.AWSCluster
		for _, cluster := range awsClusters.Items {
			if _, exists := clusterIDs[cluster.Name]; exists {
				clusterList = append(clusterList, cluster)
			}
		}
		awsClusters.Items = clusterList
	}

	return awsClusters, nil
}

func (s *Service) GetAllAWSLists(ctx context.Context, options *ListOptions) ([]runtime.Object, error) {
	var (
		err      error
		clusters []runtime.Object
	)

	var v4ClusterList *corev1alpha1.AWSClusterConfigList
	v4ClusterList, err = s.V4ListAWS(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v4ClusterList)

	var v5ClusterList *infrastructurev1alpha2.AWSClusterList
	v5ClusterList, err = s.V5ListAWS(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v5ClusterList)

	return clusters, err
}
