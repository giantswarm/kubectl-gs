package app

import (
	"context"
	"fmt"
	"net/http"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/appcatalog"

	catalogdata "github.com/giantswarm/kubectl-gs/pkg/data/domain/catalog"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid app getter Service.
type Config struct {
	Client client.Client
}

// Service is the object we'll hang the app getter methods on.
type Service struct {
	catalogDataService catalogdata.Interface
	client             client.Client
}

// New returns a new app getter Service.
func New(config Config) (*Service, error) {
	var err error

	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	var catalogDataService catalogdata.Interface
	{
		c := catalogdata.Config{
			Client: config.Client,
		}

		catalogDataService, err = catalogdata.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		client:             config.Client,
		catalogDataService: catalogDataService,
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

// Patch patches an app CR given its name and namespace.
func (s *Service) Patch(ctx context.Context, options PatchOptions) error {
	var err error

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
		appResource, err = s.getByName(ctx, namespace, name)
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

	// Make sure the requested version is available
	// Easy way:
	// (1) Reuse `catalogdata.Get(ctx, options)` to get Catalog with AppCatalogEntry CR using version-specific label selector.
	// (2) We only keep AppCatalogEntries CRs for 5 most recent versions. If step (1) returns an error, this does not necessarily
	//     mean an error. So, change selector to `latest=true` and fetch Catalog CR with AppCatalogEntry CR again. This is to reuse
	//     the `catalogdata.Get(ctx, options)` again. Catalog CR carries the URL of the given catalog, we can use it as a fallback.
	// (3) Now, fall back to checking the Helm Repository (Catalog) directly. Use HEAD request for the Chart archive, without fetching
	//     the whole index.yaml which is more "expensive".
	err = s.validateVersion(ctx, appCR.Spec.Name, version, appCR.Spec.Catalog, appCR.Spec.CatalogNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	patch := client.MergeFrom(appCR.DeepCopy())
	appCR.Spec.Version = version

	err = s.client.Patch(ctx, appCR, patch)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (s *Service) validateVersion(ctx context.Context, appName, appVersion, appCatalog, appCatalogNamespace string) error {
	selector := fmt.Sprintf(
		"application.giantswarm.io/catalog=%s,app.kubernetes.io/name=%s,app.kubernetes.io/version=%s",
		appCatalog,
		appName,
		appVersion,
	)

	/*
		(1) Check against the AppCatalogEntry CR. No error means there is an ACE CR for the given
		    version and we may stop processing here. The version is thus validated.

		    An error returned here means there is no ACE CR for the given version, but this not
		    necessarily mean the version is unavailable. Upon error we fallback to checking the
		    Helm Chart repository directly, see (2) and (3).
	*/
	_, err := s.fetchCatalog(ctx, appCatalog, appCatalogNamespace, selector)
	if err == nil {
		return nil
	}

	selector = fmt.Sprintf(
		"application.giantswarm.io/catalog=%s,app.kubernetes.io/name=%s,latest=true",
		appCatalog,
		appName,
	)

	/*
		(2) Fetch the Catalog CR and the latest AppCatalogEntry CR. We need to do it in order to get the
		    repository URL from the Catalog CR.
	*/
	catalog, err := s.fetchCatalog(ctx, appCatalog, appCatalogNamespace, selector)
	if err != nil {
		return microerror.Mask(err)
	}

	tarbalURL, err := appcatalog.NewTarballURL(catalog.Spec.Storage.URL, appName, appVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	/*
	   (3) Fallback solution, we issue a HEAD request for the Chart archive.
	*/
	// #nosec G107
	resp, err := http.Head(tarbalURL)
	if err != nil {
		return microerror.Maskf(fetchError, "unable to get the app, http request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return microerror.Mask(noResourcesError)
	}

	return nil
}

func (s *Service) fetchCatalog(ctx context.Context, name, namespace, selector string) (*applicationv1alpha1.Catalog, error) {
	var err error

	var labelSelector labels.Selector
	{
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	options := catalogdata.GetOptions{
		AllNamespaces: false,
		Name:          name,
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	catalogResource, err := s.catalogDataService.Get(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var catalogCR *applicationv1alpha1.Catalog

	switch c := catalogResource.(type) {
	case *catalogdata.Catalog:
		catalogCR = c.CR
	default:
		return nil, microerror.Maskf(invalidTypeError, "unexpected type %T found", c)
	}

	return catalogCR, nil
}

func (s *Service) getAll(ctx context.Context, namespace string) (Resource, error) {
	var err error

	appCollection := &Collection{}

	{
		lo := &client.ListOptions{
			Namespace: namespace,
		}

		apps := &applicationv1alpha1.AppList{}
		{
			err = s.client.List(ctx, apps, lo)
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
		err = s.client.Get(ctx, client.ObjectKey{
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
