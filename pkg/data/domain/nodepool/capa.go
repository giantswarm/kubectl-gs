package nodepool

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	capaexp "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllCAPA(ctx context.Context, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[capi.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	var capaMPs map[string]*capaexp.AWSMachinePool
	{
		mpCollection := &capaexp.AWSMachinePoolList{}
		err = s.client.List(ctx, mpCollection, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mpCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		capaMPs = make(map[string]*capaexp.AWSMachinePool, len(mpCollection.Items))
		for _, machinePool := range mpCollection.Items {
			mp := machinePool
			capaMPs[machinePool.GetName()] = &mp
		}
	}

	machinePools := &capiexp.MachinePoolList{}
	{
		err = s.client.List(ctx, machinePools, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(machinePools.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	npCollection := &Collection{}
	{
		for _, cr := range machinePools.Items {
			o := cr

			if capaMP, exists := capaMPs[cr.GetName()]; exists {

				np := Nodepool{
					MachinePool:     &o,
					CAPAMachinePool: capaMP,
				}
				npCollection.Items = append(npCollection.Items, np)
			}
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdCAPA(ctx context.Context, id, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		label.MachinePool: id,
	}
	if len(clusterID) > 0 {
		labelSelector[capi.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	np := &Nodepool{}

	{
		crs := &capiexp.MachinePoolList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.MachinePool = &crs.Items[0]

	}

	{
		crs := &capaexp.AWSMachinePoolList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.CAPAMachinePool = &crs.Items[0]

	}

	return np, nil
}
