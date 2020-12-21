package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAzure(ctx context.Context, namespace string) (Resource, error) {
	var err error

	options := &runtimeClient.ListOptions{
		Namespace: namespace,
	}

	var azureMPs map[string]*capzexpv1alpha3.AzureMachinePool
	{
		mpCollection := &capzexpv1alpha3.AzureMachinePoolList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, mpCollection, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mpCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		azureMPs = make(map[string]*capzexpv1alpha3.AzureMachinePool, len(mpCollection.Items))
		for _, machineDeployment := range mpCollection.Items {
			md := machineDeployment
			azureMPs[machineDeployment.GetName()] = &md
		}
	}

	machinePools := &capiexpv1alpha3.MachinePoolList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, machinePools, options)
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
					APIVersion: "exp.cluster.x-k8s.io/v1alpha3",
					Kind:       "MachinePool",
				}
				azureMP.TypeMeta = metav1.TypeMeta{
					APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1alpha3",
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

func (s *Service) getByIdAzure(ctx context.Context, id, namespace string) (Resource, error) {
	var err error

	objKey := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}

	np := &Nodepool{}

	np.MachinePool = &capiexpv1alpha3.MachinePool{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, np.MachinePool)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		np.MachinePool.TypeMeta = metav1.TypeMeta{
			APIVersion: "exp.cluster.x-k8s.io/v1alpha3",
			Kind:       "MachinePool",
		}
	}

	np.AzureMachinePool = &capzexpv1alpha3.AzureMachinePool{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, np.AzureMachinePool)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		np.AzureMachinePool.TypeMeta = metav1.TypeMeta{
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1alpha3",
			Kind:       "AzureMachinePool",
		}
	}

	return np, nil
}
