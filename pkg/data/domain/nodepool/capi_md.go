package nodepool

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllCAPI(ctx context.Context, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[capi.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	machineDeployments := &capi.MachineDeploymentList{}
	{
		err = s.client.List(ctx, machineDeployments, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(machineDeployments.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	npCollection := &Collection{}
	{
		for _, cr := range machineDeployments.Items {
			o := cr

			np := Nodepool{
				MachineDeployment: &o,
			}
			npCollection.Items = append(npCollection.Items, np)
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdCAPI(ctx context.Context, id, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		label.MachineDeployment: id,
	}
	if len(clusterID) > 0 {
		labelSelector[capi.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	np := &Nodepool{}

	{
		crs := &capi.MachineDeploymentList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.MachineDeployment = &crs.Items[0]

	}

	return np, nil
}
