package organization

import (
	"context"

	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (s *Service) Get(ctx context.Context, getOptions GetOptions) (Resource, error) {
	org, err := s.getByName(ctx, getOptions.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return org, nil
}

func (s *Service) getByName(ctx context.Context, name string) (Resource, error) {
	org := &Organization{
		Organization: &securityv1alpha1.Organization{},
	}

	key := runtimeclient.ObjectKey{
		Name:      name,
		Namespace: metav1.NamespaceNone,
	}

	err := s.client.K8sClient.CtrlClient().Get(ctx, key, org.Organization)
	if apierrors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	org.Organization.ManagedFields = nil

	org.Organization.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   securityv1alpha1.SchemeGroupVersion.Group,
		Version: securityv1alpha1.SchemeGroupVersion.Version,
		Kind:    "Organization",
	})

	return org, nil
}
