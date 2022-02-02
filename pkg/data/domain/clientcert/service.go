package clientcert

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Interface = (*Service)(nil)

type Config struct {
	Client client.Client
}

type Service struct {
	client client.Client
}

func New(config Config) (*Service, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

func (s *Service) Create(ctx context.Context, clientCert *ClientCert) error {
	kp := clientCert.Object()

	err := s.client.Create(ctx, kp)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, clientCert *ClientCert) error {
	err := s.client.Delete(ctx, clientCert.CertConfig)
	if apierrors.IsNotFound(err) {
		// Resource was already deleted.
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) GetCredential(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := s.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret)
	if apierrors.IsNotFound(err) {
		// Try in default namespace for legacy azure clusters.
		err = s.client.Get(ctx, client.ObjectKey{Name: name, Namespace: metav1.NamespaceDefault}, secret)
		if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		}
	}

	if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}
