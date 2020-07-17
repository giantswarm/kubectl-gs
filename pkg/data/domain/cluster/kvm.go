package cluster

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) v4ListKVM(ctx context.Context, namespace string) (*CommonClusterList, error) {
	var err error

	clusterConfigs := &corev1alpha1.KVMClusterConfigList{}
	{
		options := &runtimeClient.ListOptions{
			Namespace: namespace,
		}
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterConfigs, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterConfigs.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	configs := &providerv1alpha1.KVMConfigList{}
	{
		options := &runtimeClient.ListOptions{
			Namespace: namespace,
		}
		err = s.client.K8sClient.CtrlClient().List(ctx, configs, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterConfigs.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	clusters := &CommonClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
	}
	for _, cc := range clusterConfigs.Items {
		clusterConfig := cc

		var correspondingConfig runtime.Object
		{
			for _, config := range configs.Items {
				if clusterConfig.Name == fmt.Sprintf("%s-kvm-cluster-config", config.Name) {
					correspondingConfig = &config
					break
				}
			}
			if correspondingConfig == nil {
				continue
			}
		}

		newCluster := &V4ClusterList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			Items: []runtime.Object{
				&clusterConfig,
				correspondingConfig,
			},
		}

		clusters.Items = append(clusters.Items, newCluster)
	}

	return clusters, nil
}

func (s *Service) getAllKVM(ctx context.Context, namespace string) ([]runtime.Object, error) {
	var (
		err      error
		clusters []runtime.Object
	)

	v4ClusterList, err := s.v4ListKVM(ctx, namespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	clusters = append(clusters, v4ClusterList.Items...)

	return clusters, err
}

func (s *Service) v4GetByIdKVM(ctx context.Context, id, namespace string) (*V4ClusterList, error) {
	var err error

	clusterConfig := &corev1alpha1.KVMClusterConfig{}
	{
		key := runtimeClient.ObjectKey{
			Name:      fmt.Sprintf("%s-kvm-cluster-config", id),
			Namespace: namespace,
		}
		err = s.client.K8sClient.CtrlClient().Get(ctx, key, clusterConfig)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	config := &providerv1alpha1.KVMConfig{}
	{
		key := runtimeClient.ObjectKey{
			Name:      id,
			Namespace: namespace,
		}
		err = s.client.K8sClient.CtrlClient().Get(ctx, key, config)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	v4ClusterList := &V4ClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: []runtime.Object{
			clusterConfig,
			config,
		},
	}

	return v4ClusterList, nil
}

func (s *Service) getByIdKVM(ctx context.Context, id, namespace string) (runtime.Object, error) {
	cluster, err := s.v4GetByIdKVM(ctx, id, namespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cluster, nil
}
