package nodepool

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) ([]Nodepool, error) {
	var err error

	options := &runtimeClient.ListOptions{
		Namespace: namespace,
	}

	var awsMDs map[string]*infrastructurev1alpha2.AWSMachineDeployment
	{
		mdCollection := &infrastructurev1alpha2.AWSMachineDeploymentList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, mdCollection, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mdCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		awsMDs = make(map[string]*infrastructurev1alpha2.AWSMachineDeployment, len(mdCollection.Items))
		for _, machineDeployment := range mdCollection.Items {
			md := machineDeployment
			awsMDs[machineDeployment.GetName()] = &md
		}
	}

	machineDeployments := &capiv1alpha2.MachineDeploymentList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, machineDeployments, options)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(machineDeployments.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	var npCollection []Nodepool
	{
		for _, cr := range machineDeployments.Items {
			o := cr

			if awsMD, exists := awsMDs[cr.GetName()]; exists {
				cr.TypeMeta = metav1.TypeMeta{
					APIVersion: "cluster.x-k8s.io/v1alpha2",
					Kind:       "MachineDeployment",
				}
				awsMD.TypeMeta = infrastructurev1alpha2.NewAWSMachineDeploymentTypeMeta()

				np := Nodepool{
					MachineDeployment:    &o,
					AWSMachineDeployment: awsMD,
				}
				npCollection = append(npCollection, np)
			}
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdAWS(ctx context.Context, id, namespace string) (Nodepool, error) {
	var err error
	var np Nodepool

	objKey := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}

	np.MachineDeployment = &capiv1alpha2.MachineDeployment{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, np.MachineDeployment)
		if errors.IsNotFound(err) {
			return Nodepool{}, microerror.Mask(notFoundError)
		} else if err != nil {
			return Nodepool{}, microerror.Mask(err)
		}
	}

	np.AWSMachineDeployment = &infrastructurev1alpha2.AWSMachineDeployment{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, np.AWSMachineDeployment)
		if errors.IsNotFound(err) {
			return Nodepool{}, microerror.Mask(notFoundError)
		} else if err != nil {
			return Nodepool{}, microerror.Mask(err)
		}

		np.AWSMachineDeployment.TypeMeta = infrastructurev1alpha2.NewAWSMachineDeploymentTypeMeta()
	}

	return np, nil
}
