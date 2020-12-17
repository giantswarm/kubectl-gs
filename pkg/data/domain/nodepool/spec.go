package nodepool

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	capiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

type GetOptions struct {
	ID        string
	Provider  string
	Namespace string
}

type Interface interface {
	Get(context.Context, GetOptions) (NodepoolCollection, error)
}

type Nodepool struct {
	MachineDeployment    *capiv1alpha2.MachineDeployment
	AWSMachineDeployment *infrastructurev1alpha2.AWSMachineDeployment
}

type NodepoolCollection struct {
	Items []Nodepool
}
