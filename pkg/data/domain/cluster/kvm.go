package cluster

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
)

func (s *Service) V4ListKVM(ctx context.Context, options *ListOptions) (*corev1alpha1.KVMClusterConfigList, error) {
	clusters := &corev1alpha1.KVMClusterConfigList{}
	err := s.client.K8sClient.CtrlClient().List(ctx, clusters)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(clusters.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, nil
}

func (s *Service) GetAllKVMLists(ctx context.Context, options *ListOptions) ([]runtime.Object, error) {
	var (
		err      error
		clusters []runtime.Object
	)

	var v4ClusterList *corev1alpha1.KVMClusterConfigList
	v4ClusterList, err = s.V4ListKVM(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v4ClusterList)

	return clusters, err
}
