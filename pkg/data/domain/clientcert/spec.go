package clientcert

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ClientCert struct {
	CertConfig *corev1alpha1.CertConfig
}

type Collection struct {
	Items []ClientCert
}

type Resource interface {
	Object() runtime.Object
}

type Interface interface {
	Create(ctx context.Context, clientCert *ClientCert) error
	Delete(ctx context.Context, clientCert *ClientCert) error
	GetCredential(ctx context.Context, namespace, name string) (*corev1.Secret, error)
}

func (k *ClientCert) Object() runtime.Object {
	if k.CertConfig != nil {
		return k.CertConfig
	}

	return nil
}

func (c *Collection) Object() runtime.Object {
	list := &metav1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}

	for _, item := range c.Items {
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
