package app

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Get fetches a list of app CRs filtered by namespace and optionally by
// name.
func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Namespace, options.Name)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		return resource, nil
	}

	resource, err = s.getAll(ctx, options.Namespace)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return resource, nil
}

func (s *Service) Patch(ctx context.Context, options PatchOptions) error {
	var err error

	// is passing version or other properties for that matter, through the `options` considered good?
	// if not what would be a good way to pass the new parameters and yet make the function generic?
	// would it allow extending the update sub-command with another patches?
	if len(options.Version) > 0 {
		err = s.patchVersion(ctx, options.Namespace, options.Name, options.Version)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	return nil
}

func (s *Service) patchVersion(ctx context.Context, namespace string, name string, version string) error {
	var err error

	var appResource Resource
	{
		// reuse the getByName ??
		appResource, _ = s.getByName(ctx, namespace, name)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appCR *applicationv1alpha1.App

	switch a := appResource.(type) {
	case *App:
		appCR = a.CR
	default:
		return microerror.Maskf(invalidTypeError, "unexpected type %T found", a)
	}

	patch := runtimeclient.MergeFrom(appCR.DeepCopy())
	appCR.Spec.Version = version

	err = s.client.K8sClient.CtrlClient().Patch(ctx, appCR, patch)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
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
			if apimeta.IsNoMatchError(err) {
				return nil, microerror.Mask(noMatchError)
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else if len(apps.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, app := range apps.Items {
			a := App{
				CR: omitManagedFields(app.DeepCopy()),
			}
			appCollection.Items = append(appCollection.Items, a)
		}
	}

	return appCollection, nil
}

func (s *Service) getByName(ctx context.Context, namespace, name string) (Resource, error) {
	var err error

	app := &App{}
	{
		appCR := &applicationv1alpha1.App{}
		err = s.client.K8sClient.CtrlClient().Get(ctx, runtimeclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, appCR)
		if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if apimeta.IsNoMatchError(err) {
			return nil, microerror.Mask(noMatchError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		app.CR = omitManagedFields(appCR)
		app.CR.TypeMeta = metav1.TypeMeta{
			APIVersion: "app.application.giantswarm.io/v1alpha1",
			Kind:       "App",
		}
	}

	return app, nil
}

// omitManagedFields removes managed fields to make YAML output easier to read.
// With Kubernetes 1.21 we can use OmitManagedFieldsPrinter and remove this.
func omitManagedFields(app *applicationv1alpha1.App) *applicationv1alpha1.App {
	app.ManagedFields = nil
	return app
}
