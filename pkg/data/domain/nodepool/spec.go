package nodepool

import (
	"context"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capaexp "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

type GetOptions struct {
	Name        string
	ClusterName string
	Provider    string
	Namespace   string
}

type Interface interface {
	Get(context.Context, GetOptions) (Resource, error)
}

type Resource interface {
	Object() runtime.Object
}

// Nodepool abstracts away provider-specific
// node pool resources.
type Nodepool struct {
	MachineDeployment *capi.MachineDeployment
	MachinePool       *capiexp.MachinePool

	AWSMachineDeployment  *infrastructurev1alpha3.AWSMachineDeployment
	AzureMachinePool      *capzexp.AzureMachinePool
	CAPAMachinePool       *capaexp.AWSMachinePool
	EKSManagedMachinePool *capaexp.AWSManagedMachinePool
}

func (n *Nodepool) Object() runtime.Object {
	if n.MachineDeployment != nil {
		return n.MachineDeployment
	} else if n.MachinePool != nil {
		return n.MachinePool
	}

	return nil
}

// Collection wraps a list of nodepools.
type Collection struct {
	Items []Nodepool
}

func (nc *Collection) Object() runtime.Object {
	list := &metav1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}

	for _, item := range nc.Items {
		obj := item.Object()
		if obj == nil {
			continue
		}

		raw := runtime.RawExtension{
			Object: obj,
		}
		list.Items = append(list.Items, raw)
	}

	return list
}
