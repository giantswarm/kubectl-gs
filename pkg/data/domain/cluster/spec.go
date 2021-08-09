package cluster

import (
	"context"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

type GetOptions struct {
	Name      string
	Provider  string
	Namespace string
}

// Interface represents the contract for the clusters service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Get(context.Context, GetOptions) (Resource, error)
}

type Resource interface {
	Object() runtime.Object
}

// Cluster abstracts away provider-specific
// node pool resources.
type Cluster struct {
	Cluster *capiv1alpha3.Cluster

	AWSCluster   *infrastructurev1alpha3.AWSCluster
	AzureCluster *capzv1alpha3.AzureCluster
}

func (n *Cluster) Object() runtime.Object {
	if n.Cluster != nil {
		return n.Cluster
	}

	return nil
}

// Collection wraps a list of clusters.
type Collection struct {
	Items []Cluster
}

func (cc *Collection) Object() runtime.Object {
	list := &metav1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}

	for _, item := range cc.Items {
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
