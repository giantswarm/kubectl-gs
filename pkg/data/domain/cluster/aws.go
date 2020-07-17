package cluster

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) v4ListAWS(ctx context.Context) (*CommonClusterList, error) {
	var err error

	clusterConfigs := &corev1alpha1.AWSClusterConfigList{}
	{
		options := &runtimeClient.ListOptions{
			Namespace: "default",
		}
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterConfigs, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterConfigs.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	configs := &providerv1alpha1.AWSConfigList{}
	{
		options := &runtimeClient.ListOptions{
			Namespace: "default",
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
				if clusterConfig.Name == fmt.Sprintf("%s-aws-cluster-config", config.Name) {
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

func (s *Service) v5ListAWS(ctx context.Context) (*infrastructurev1alpha2.AWSClusterList, error) {
	clusterIDs, err := s.getClusterIDs(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	awsClusters := &infrastructurev1alpha2.AWSClusterList{}
	options := &runtimeClient.ListOptions{
		Namespace: "default",
	}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, awsClusters, options)
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

func (s *Service) getAllAWS(ctx context.Context) ([]runtime.Object, error) {
	var (
		err      error
		clusters []runtime.Object
	)

	v5ClusterList, err := s.v5ListAWS(ctx)
	if err == nil {
		for _, c := range v5ClusterList.Items {
			clusters = append(clusters, &c)
		}
	} else if IsNoResources(err) {
		// Fall through.
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	v4ClusterList, err := s.v4ListAWS(ctx)
	if err == nil {
		clusters = append(clusters, v4ClusterList.Items...)
	} else if IsNoResources(err) {
		// Fall through.
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(clusters) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return clusters, err
}

func (s *Service) v4GetByIdAWS(ctx context.Context, id string) (*V4ClusterList, error) {
	var err error

	clusterConfig := &corev1alpha1.AWSClusterConfig{}
	{
		key := runtimeClient.ObjectKey{
			Name:      fmt.Sprintf("%s-aws-cluster-config", id),
			Namespace: "default",
		}
		err = s.client.K8sClient.CtrlClient().Get(ctx, key, clusterConfig)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	config := &providerv1alpha1.AWSConfig{}
	{
		key := runtimeClient.ObjectKey{
			Name:      id,
			Namespace: "default",
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

	cluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()

	return cluster, nil
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
