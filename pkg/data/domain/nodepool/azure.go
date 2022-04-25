package nodepool

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAzure(ctx context.Context, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[capiv1beta1.ClusterLabelName] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	var azureMPs map[string]*capzexp.AzureMachinePool
	{
		mpCollection := &capzexp.AzureMachinePoolList{}
		err = s.client.List(ctx, mpCollection, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mpCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		azureMPs = make(map[string]*capzexp.AzureMachinePool, len(mpCollection.Items))
		for _, machineDeployment := range mpCollection.Items {
			md := machineDeployment
			azureMPs[machineDeployment.GetName()] = &md
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

			if azureMP, exists := azureMPs[cr.GetName()]; exists {
				cr.TypeMeta = metav1.TypeMeta{
					APIVersion: "exp.cluster.x-k8s.io/v1beta1",
					Kind:       "MachinePool",
				}
				azureMP.TypeMeta = metav1.TypeMeta{
					APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "AzureMachinePool",
				}

				np := Nodepool{
					MachinePool:      &o,
					AzureMachinePool: azureMP,
				}
				npCollection.Items = append(npCollection.Items, np)
			}
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdAzure(ctx context.Context, id, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		label.MachinePool: id,
	}
	if len(clusterID) > 0 {
		labelSelector[capiv1beta1.ClusterLabelName] = clusterID
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

		np.MachinePool.TypeMeta = metav1.TypeMeta{
			APIVersion: "exp.cluster.x-k8s.io/v1beta1",
			Kind:       "MachinePool",
		}
	}

	{
		crs := &capzexp.AzureMachinePoolList{}
		err = s.client.List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.AzureMachinePool = &crs.Items[0]

		np.AzureMachinePool.TypeMeta = metav1.TypeMeta{
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureMachinePool",
		}
	}

	return np, nil
}
