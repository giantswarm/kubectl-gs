package nodepool

import (
	"context"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{}
	if len(clusterID) > 0 {
		labelSelector[label.Cluster] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	var awsMDs map[string]*infrastructurev1alpha3.AWSMachineDeployment
	{
		mdCollection := &infrastructurev1alpha3.AWSMachineDeploymentList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, mdCollection, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(mdCollection.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}

		awsMDs = make(map[string]*infrastructurev1alpha3.AWSMachineDeployment, len(mdCollection.Items))
		for _, machineDeployment := range mdCollection.Items {
			md := machineDeployment
			awsMDs[machineDeployment.GetName()] = &md
		}
	}

	machineDeployments := &capiv1alpha3.MachineDeploymentList{}
	{
		err = s.client.K8sClient.CtrlClient().List(ctx, machineDeployments, labelSelector, inNamespace)
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

			if awsMD, exists := awsMDs[cr.GetName()]; exists {
				cr.TypeMeta = metav1.TypeMeta{
					APIVersion: "cluster.x-k8s.io/v1alpha3",
					Kind:       "MachineDeployment",
				}
				awsMD.TypeMeta = infrastructurev1alpha3.NewAWSMachineDeploymentTypeMeta()

				np := Nodepool{
					MachineDeployment:    &o,
					AWSMachineDeployment: awsMD,
				}
				npCollection.Items = append(npCollection.Items, np)
			}
		}
	}

	return npCollection, nil
}

func (s *Service) getByIdAWS(ctx context.Context, id, namespace, clusterID string) (Resource, error) {
	var err error

	labelSelector := runtimeClient.MatchingLabels{
		label.MachineDeployment: id,
	}
	if len(clusterID) > 0 {
		labelSelector[label.Cluster] = clusterID
	}
	inNamespace := runtimeClient.InNamespace(namespace)

	np := &Nodepool{}

	{
		crs := &capiv1alpha3.MachineDeploymentList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.MachineDeployment = &crs.Items[0]

		np.MachineDeployment.TypeMeta = metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha3",
			Kind:       "MachineDeployment",
		}
	}

	{
		crs := &infrastructurev1alpha3.AWSMachineDeploymentList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, crs, labelSelector, inNamespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(crs.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		np.AWSMachineDeployment = &crs.Items[0]

		np.AWSMachineDeployment.TypeMeta = infrastructurev1alpha3.NewAWSMachineDeploymentTypeMeta()
	}

	return np, nil
}
