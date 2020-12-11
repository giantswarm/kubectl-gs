package nodepool

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) getAllAWS(ctx context.Context, namespace string) (runtime.Object, error) {
	var err error

	options := &runtimeClient.ListOptions{
		Namespace: namespace,
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

	var mdCollection runtime.Object
	{
		var mds []runtime.Object
		for _, cr := range machineDeployments.Items {
			r := cr

			if awsMD, exists := awsMDs[cr.GetName()]; exists {
				md := []runtime.Object{
					&r,
					awsMD,
				}
				mds = append(mds, toV1List(md))
			}
		}

		mdCollection = toV1List(mds)
	}

	return mdCollection, nil
}

func (s *Service) getByIdAWS(ctx context.Context, id, namespace string) (*infrastructurev1alpha2.AWSCluster, error) {
	var err error

	objKey := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: namespace,
	}

	// The CAPI cluster has the labels. It can be used for filtering.
	cluster := &capiv1alpha2.Cluster{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, cluster)
		if errors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	awsCluster := &infrastructurev1alpha2.AWSCluster{}
	err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, awsCluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	awsCluster.TypeMeta = infrastructurev1alpha2.NewAWSClusterTypeMeta()

	return awsCluster, nil
}
