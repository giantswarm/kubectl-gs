package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
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

	{
		cluster.APIVersion = "v1alpha3"
		cluster.Kind = "Cluster"
	}

	return cluster, nil
}
