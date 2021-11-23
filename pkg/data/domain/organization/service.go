package organization

import (
	"context"

	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	var resource Resource
	var err error

	if len(getOptions.Name) > 0 {
		resource, err = s.getByName(ctx, getOptions.Name)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
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

	return org, nil
}

func (s *Service) getAll(ctx context.Context) (Resource, error) {
	var err error

	orgCollection := &Collection{}
	{
		orgs := &securityv1alpha1.OrganizationList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, orgs)
			if err != nil {
				return nil, microerror.Mask(err)
			} else if len(orgs.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, org := range orgs.Items {
			o := Organization{
				Organization: org.DeepCopy(),
			}
			o.Organization.ManagedFields = nil

			orgCollection.Items = append(orgCollection.Items, o)
		}
	}

	return orgCollection, nil
}
