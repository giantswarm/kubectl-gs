package nodepool

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func (n *Nodepool) Object() runtime.Object {
	if n.MachineDeployment != nil {
		return n.MachineDeployment
	}

	return nil
}

type NodepoolCollection struct {
	Items []Nodepool
}

func (nc *NodepoolCollection) List() *metav1.List {
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
