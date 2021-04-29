package app

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
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

// Get fetches an App CR by name and namespace.
func (s *Service) Get(ctx context.Context, options GetOptions) (*applicationv1alpha1.App, error) {
	namespacedName := types.NamespacedName{
		Namespace: options.Namespace,
		Name:      options.Name,
	}

	app := &applicationv1alpha1.App{}
	err := s.client.K8sClient.CtrlClient().Get(ctx, namespacedName, app)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return app, nil
}

// GetObject fetches a list of app CRs filtered by namespace and optionally by
// name.
func (s *Service) GetObject(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Name, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

// List returns a list of App CRs in a namespace, optionally filtered by a label selector.
func (s *Service) List(ctx context.Context, options ListOptions) (*applicationv1alpha1.AppList, error) {
	parsedSelector, err := labels.Parse(options.LabelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	o := &runtimeclient.ListOptions{
		Namespace:     options.Namespace,
		LabelSelector: parsedSelector,
	}

	appList := &applicationv1alpha1.AppList{}
	err = s.client.K8sClient.CtrlClient().List(ctx, appList, o)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return appList, nil

}

func (s *Service) getAll(ctx context.Context, namespace string) (Resource, error) {
	var err error

	appCollection := &Collection{}

	{
		lo := &runtimeclient.ListOptions{
			Namespace: namespace,
		}

		apps := &applicationv1alpha1.AppList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, apps, lo)
			if err != nil {
				return nil, microerror.Mask(err)
			} else if len(apps.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, app := range apps.Items {
			a := App{
				CR: app.DeepCopy(),
			}
			appCollection.Items = append(appCollection.Items, a)
		}
	}

	return appCollection, nil
}

func (s *Service) getByName(ctx context.Context, name, namespace string) (Resource, error) {
	var err error

	appCollection := &Collection{}

	{
		lo := &runtimeclient.ListOptions{
			Namespace: namespace,
		}

		apps := &applicationv1alpha1.AppList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, apps, lo)
			if err != nil {
				return nil, microerror.Mask(err)
			} else if len(apps.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, app := range apps.Items {
			a := App{
				CR: app.DeepCopy(),
			}
			appCollection.Items = append(appCollection.Items, a)
		}
	}

	return appCollection, nil
}
