package cluster

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/internal/label"
)

func (s *Service) getAllAzure(ctx context.Context, namespace string) (*capiv1alpha3.ClusterList, error) {
	var err error

	options := &runtimeClient.ListOptions{
		Namespace: namespace,
	}

	clusterList := &capiv1alpha3.ClusterList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, clusterList, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(clusterList.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	var objKey runtimeClient.ObjectKey
	for _, cluster := range clusterList.Items {
		// FIXME(axbarsan): Remove once the name is stored in
		// an annotation, rather than a config map.
		var config corev1.ConfigMap
		{
			objKey = runtimeClient.ObjectKey{
				Name:      fmt.Sprintf("%s-cluster-user-values", cluster.Name),
				Namespace: cluster.Namespace,
			}
			err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, &config)
			if errors.IsNotFound(err) {
				// Fall through.
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else {
				if cluster.Annotations == nil {
					cluster.Annotations = make(map[string]string)
				}
				cluster.Annotations[label.Description] = config.Data["cluster.description"]
			}
		}
	}

	{
		clusterList.APIVersion = "v1"
		clusterList.Kind = "List"
	}

	return clusterList, nil
}

func (s *Service) getByIdAzure(ctx context.Context, id, namespace string) (*capiv1alpha3.Cluster, error) {
	var (
		err    error
		objKey runtimeClient.ObjectKey
	)

	objKey = runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}
	cluster := &capiv1alpha3.Cluster{}
	err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, cluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	// FIXME(axbarsan): Remove once the name is stored in
	// an annotation, rather than a config map.
	var config corev1.ConfigMap
	{
		objKey = runtimeClient.ObjectKey{
			Name:      fmt.Sprintf("%s-cluster-user-values", id),
			Namespace: namespace,
		}
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, &config)
		if errors.IsNotFound(err) {
			// Fall through.
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			if cluster.Annotations == nil {
				cluster.Annotations = make(map[string]string)
			}
			cluster.Annotations[label.Description] = config.Data["cluster.description"]
		}
	}

	{
		cluster.APIVersion = "v1alpha3"
		cluster.Kind = "Cluster"
	}

	return cluster, nil
}
