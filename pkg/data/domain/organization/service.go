package organization

import (
	"context"

	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

type Config struct {
	Client *client.Client
}

type Service struct {
	client *client.Client
}

func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Name)
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

func (s *Service) getAll(ctx context.Context) (Resource, error) {
	var err error

	organizations := &securityv1alpha1.OrganizationList{}
	{
		opts := runtimeClient.ListOptions{Namespace: ""}
		err = s.client.K8sClient.CtrlClient().List(ctx, organizations, &opts)
		if err != nil {
			return nil, microerror.Mask(err)
		} else if len(organizations.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	organizationCollection := &Collection{}
	{
		for i := range organizations.Items {
			o := organizations.Items[i]
			org := Organization{
				Organization: &o,
			}

			organizationCollection.Items = append(organizationCollection.Items, org)
		}
	}

	return organizationCollection, nil
}

func (s *Service) getByName(ctx context.Context, name string) (Resource, error) {
	var err error

	org := &Organization{}
	{
		organizations := &securityv1alpha1.OrganizationList{}
		err = s.client.K8sClient.CtrlClient().List(ctx, organizations)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if len(organizations.Items) < 1 {
			return nil, microerror.Mask(notFoundError)
		}
		org.Organization = &organizations.Items[0]
	}

	return org, nil
}
