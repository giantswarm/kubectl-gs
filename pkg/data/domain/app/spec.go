package app

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// App abstracts away the custom resource so it can be returned as a runtime
// object or a typed custom resource.
type App struct {
	CR *applicationv1alpha1.App
}

// Collection wraps a list of apps.
type Collection struct {
	Items []App
}

// GetOptions are the parameters that the Get method takes.
type GetOptions struct {
	LabelSelector string
	Name          string
	Namespace     string
}

// PatchOptions are the parameters that the Patch method takes.
type PatchOptions struct {
	Name                  string
	Namespace             string
	SuspendReconciliation bool
	Version               string
}

type Resource interface {
	Object() runtime.Object
}

// Interface represents the contract for the app data service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Get(context.Context, GetOptions) (Resource, error)
	Patch(context.Context, PatchOptions) error
}

func (a *App) Object() runtime.Object {
	if a.CR != nil {
		return a.CR
	}

	return nil
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
