package release

import (
	"context"

	releasev1alpha1 "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type GetOptions struct {
	Name       string
	ActiveOnly bool
	Provider   string
	Namespace  string
}

type Interface interface {
	Get(context.Context, GetOptions) (Resource, error)
}

type Resource interface {
	Object() runtime.Object
}

// Release abstracts away the custom resource so it can be returned as a runtime
// object or a typed custom resource.
type Release struct {
	CR *releasev1alpha1.Release
}

func (r *Release) Object() runtime.Object {
	if r.CR != nil {
		return r.CR
	}

	return nil
}

// ReleaseCollections wraps a list of releases.
type ReleaseCollection struct {
	Items []Release
}

func (cc *ReleaseCollection) Object() runtime.Object {
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
