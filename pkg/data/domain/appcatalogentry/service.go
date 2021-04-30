package appcatalogentry

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid app getter Service.
type Config struct {
	Client *client.Client
}

// Service is the object we'll hang the app getter methods on.
type Service struct {
	client *client.Client
}

// New returns a new app getter Service.
func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

// GetObject fetches a list of app CRs filtered by namespace and optionally by
// name.
func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.App) > 0 {
		resource, err = s.getByName(ctx, options.App, options.Catalog)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Catalog)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getAll(ctx context.Context, catalog string) (Resource, error) {
	var err error

	entriesCollection := &Collection{}

	{
		lo := &runtimeclient.ListOptions{}
		l := &applicationv1alpha1.AppCatalogEntryList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, l, lo)
			if err != nil {
				return nil, microerror.Mask(err)
			} else if len(l.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, ace := range l.Items {
			a := AppCatalogEntry{
				CR: ace.DeepCopy(),
			}
			entriesCollection.Items = append(entriesCollection.Items, a)
		}
	}

	return entriesCollection, nil
}

func (s *Service) getByName(ctx context.Context, app, catalog string) (Resource, error) {
	var err error

	entriesCollection := &Collection{}

	{
		lo := &runtimeclient.ListOptions{}
		l := &applicationv1alpha1.AppCatalogEntryList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, l, lo)
			if err != nil {
				return nil, microerror.Mask(err)
			} else if len(l.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, ace := range l.Items {
			a := AppCatalogEntry{
				CR: ace.DeepCopy(),
			}
			entriesCollection.Items = append(entriesCollection.Items, a)
		}
	}

	return entriesCollection, nil
}
