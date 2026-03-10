package nodepool

import (
	"context"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v6/internal/key"
)

func (s *Service) getAllCAPI(ctx context.Context, namespace, clusterID string) (Resource, error) {
	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[key.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	machineDeployments := &unstructured.UnstructuredList{}
	machineDeployments.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "MachineDeploymentList",
	})

	err := s.client.List(ctx, machineDeployments, labelSelector, inNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(machineDeployments.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	npCollection := &Collection{}
	for i := range machineDeployments.Items {
		item := &machineDeployments.Items[i]
		np := Nodepool{
			MachineDeployment: item,
		}
		npCollection.Items = append(npCollection.Items, np)
	}

	return npCollection, nil
}

func (s *Service) getByIdCAPI(ctx context.Context, id, namespace, clusterID string) (Resource, error) {
	labelSelector := runtimeClient.MatchingLabels{
		label.MachineDeployment: id,
	}
	if len(clusterID) > 0 {
		labelSelector[key.ClusterNameLabel] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	np := &Nodepool{}

	machineDeployments := &unstructured.UnstructuredList{}
	machineDeployments.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "MachineDeploymentList",
	})

	err := s.client.List(ctx, machineDeployments, labelSelector, inNamespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(machineDeployments.Items) < 1 {
		return nil, microerror.Mask(notFoundError)
	}
	np.MachineDeployment = &machineDeployments.Items[0]

	return np, nil
}
