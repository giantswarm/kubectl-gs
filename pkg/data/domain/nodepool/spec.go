package nodepool

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	MachineDeployment *unstructured.Unstructured
	MachinePool       *unstructured.Unstructured

	CAPAMachinePool       *unstructured.Unstructured
	EKSManagedMachinePool *unstructured.Unstructured
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
