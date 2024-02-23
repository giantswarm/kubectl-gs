package nodepool

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	capaexp "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllCAPI(ctx context.Context, namespace, clusterID, provider string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[capi.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	machinePools := &capiexp.MachinePoolList{}
	{
		err = s.client.List(ctx, machinePools, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(machinePools.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	infraMPs := make(map[string]interface{})

	{
		mpCollection := &unstructured.UnstructuredList{}
		switch provider {
		case key.ProviderCAPA:
			mpCollection.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta1")
			mpCollection.SetKind("AWSMachinePoolList")
		case key.ProviderCAPZ:
			mpCollection.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta1")
			mpCollection.SetKind("AzureMachinePoolList")
		}

		err = s.client.List(ctx, mpCollection, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mpCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		for _, machinePool := range mpCollection.Items {
			mp := machinePool
			switch provider {
			case key.ProviderCAPA:
				capaMP := &capaexp.AWSMachinePool{}
				err := s.client.Scheme().Convert(&mp, capaMP, nil)
				if err != nil {
					return nil, microerror.Mask(err)
				}
				infraMPs[machinePool.GetName()] = capaMP
			case key.ProviderCAPZ:
				capzMP := &capzexp.AzureMachinePool{}
				err := s.client.Scheme().Convert(&mp, capzMP, nil)
				if err != nil {
					return nil, microerror.Mask(err)
				}
				infraMPs[machinePool.GetName()] = capzMP
			}
		}
	}

	npCollection := &Collection{
		Items: []Nodepool{},
	}
	for _, cr := range machinePools.Items {
		o := cr

		if infraMP, exists := infraMPs[cr.GetName()]; exists {
			cr.TypeMeta = metav1.TypeMeta{
				APIVersion: "exp.cluster.x-k8s.io/v1beta1",
				Kind:       "MachinePool",
			}
			np := Nodepool{
				MachinePool: &o,
			}
			switch provider {
			case key.ProviderCAPA:
				capaMP := infraMP.(*capaexp.AWSMachinePool)
				capaMP.TypeMeta = metav1.TypeMeta{
					APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "AWSMachinePool",
				}
				np.CAPAMachinePool = capaMP
			case key.ProviderCAPZ:
				capzMP := infraMP.(*capzexp.AzureMachinePool)
				capzMP.TypeMeta = metav1.TypeMeta{
					APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "AzureMachinePool",
				}
				np.AzureMachinePool = capzMP
			}

			npCollection.Items = append(npCollection.Items, np)
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdCAPI(ctx context.Context, id, namespace, clusterID, provider string) (Resource, error) {
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

		np.MachinePool.TypeMeta = metav1.TypeMeta{
			APIVersion: "exp.cluster.x-k8s.io/v1beta1",
			Kind:       "MachinePool",
		}
	}

	{
		mp := &unstructured.UnstructuredList{}

		switch provider {
		case key.ProviderCAPA:
			mp.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta1")
			mp.SetKind("AWSMachinePoolList")
		case key.ProviderCAPZ:
			mp.SetAPIVersion("infrastructure.cluster.x-k8s.io/v1beta1")
			mp.SetKind("AzureMachinePoolList")
		}
		err = s.client.List(ctx, mp, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mp.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		o := mp.Items[0]
		switch provider {
		case key.ProviderCAPA:
			capaMP := &capaexp.AWSMachinePool{}
			err := s.client.Scheme().Convert(&o, capaMP, nil)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			np.CAPAMachinePool = capaMP
		case key.ProviderCAPZ:
			capzMP := &capzexp.AzureMachinePool{}
			err := s.client.Scheme().Convert(&o, capzMP, nil)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			np.AzureMachinePool = capzMP
		}

	}

	return np, nil
}
