package cluster

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) v4ListAWS(ctx context.Context) (*corev1alpha1.AWSClusterConfigList, error) {
	clusters := &corev1alpha1.AWSClusterConfigList{}
	options := &runtimeClient.ListOptions{
		Namespace: "default",
	}
	err := s.client.K8sClient.CtrlClient().List(ctx, clusters, options)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, nil
}

func (s *Service) v4GetByIdAWS(ctx context.Context, id string) (*corev1alpha1.AWSClusterConfig, error) {
	cluster := &corev1alpha1.AWSClusterConfig{}
	key := runtimeClient.ObjectKey{
		Name:      fmt.Sprintf("%s-aws-cluster-config", id),
		Namespace: "default",
	}
	err := s.client.K8sClient.CtrlClient().Get(ctx, key, cluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}

func (s *Service) v5ListAWS(ctx context.Context) (*infrastructurev1alpha2.AWSClusterList, error) {
	clusterIDs, err := s.getClusterIDs(ctx)
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

func (s *Service) v5GetByIdAWS(ctx context.Context, id string) (*infrastructurev1alpha2.AWSCluster, error) {
	cluster := &infrastructurev1alpha2.AWSCluster{}
	key := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: "default",
	}
	err := s.client.K8sClient.CtrlClient().Get(ctx, key, cluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}

func (s *Service) getAllListsAWS(ctx context.Context) ([]runtime.Object, error) {
	var (
		err      error
		clusters []runtime.Object
	)

	var v4ClusterList *corev1alpha1.AWSClusterConfigList
	v4ClusterList, err = s.v4ListAWS(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v4ClusterList)

	var v5ClusterList *infrastructurev1alpha2.AWSClusterList
	v5ClusterList, err = s.v5ListAWS(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v5ClusterList)

	return clusters, err
}

func (s *Service) getByIdAWS(ctx context.Context, id string) (runtime.Object, error) {
	var (
		err     error
		cluster runtime.Object
	)

	cluster, err = s.v5GetByIdAWS(ctx, id)
	if err == nil {
		return cluster, nil
	} else if IsNotFound(err) {
		// Fall through, try v4.
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	cluster, err = s.v4GetByIdAWS(ctx, id)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}
