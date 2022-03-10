package cluster

import (
	"context"

	application "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	infrastructure "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
)

type GetOptions struct {
	Name      string
	Provider  string
	Namespace string
}

type PatchOptions struct {
	PatchSpecs []PatchSpec
}

type PatchSpec struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// Interface represents the contract for the clusters service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Get(context.Context, GetOptions) (Resource, error)
	Patch(context.Context, runtime.Object, PatchOptions) error
}

type Resource interface {
	Object() runtime.Object
}

// Cluster contains the resources needed to represent a cluster on any supported provider.
type Cluster struct {
	Cluster *capi.Cluster

	// helm-based clusters
	ClusterApp     *application.App
	DefaultAppsApp *application.App

	// infrastructure provider cluster
	AWSCluster   *infrastructure.AWSCluster
	AzureCluster *capz.AzureCluster
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
	list := &meta.List{
		TypeMeta: meta.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: meta.ListMeta{},
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
