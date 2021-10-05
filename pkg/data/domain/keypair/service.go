package keypair

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = (*Service)(nil)

type Config struct {
	Client *client.Client
}

type Service struct {
	client *client.Client
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

func (s *Service) Create(ctx context.Context, keypair *Keypair) error {
	kp := keypair.Object()

	err := s.client.K8sClient.CtrlClient().Create(ctx, kp)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, keypair *Keypair) error {
	kp := keypair.Object()
	namespace := runtimeclient.InNamespace(metav1.NamespaceDefault)

	err := s.client.K8sClient.CtrlClient().DeleteAllOf(ctx, kp, namespace)
	if apierrors.IsNotFound(err) {
		// Resource was already deleted.
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) GetCredential(ctx context.Context, name string) (*corev1.Secret, error) {
	secret, err := s.client.K8sClient.K8sClient().CoreV1().Secrets(metav1.NamespaceDefault).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}
