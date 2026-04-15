package clientcert

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientCert struct {
	CertConfig *corev1alpha1.CertConfig
}

type Resource interface {
	Object() client.Object
}

type Interface interface {
	Create(ctx context.Context, clientCert *ClientCert) error
	Delete(ctx context.Context, clientCert *ClientCert) error
	GetCredential(ctx context.Context, namespace, name string) (*corev1.Secret, error)
}

func (k *ClientCert) Object() client.Object {
	if k.CertConfig != nil {
		return k.CertConfig
	}

	return nil
}
