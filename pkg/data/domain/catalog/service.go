package catalog

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid catalog getter Service.
type Config struct {
	Client client.Client
}

// Service is the object we'll hang the catalog getter methods on.
type Service struct {
	client client.Client
}

// New returns a new catalog getter Service.
func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

// Get fetches a list of catalog CRs optionally filtered by name.
func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Namespace, options.Name, options.LabelSelector)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		return resource, nil
	}

	resource, err = s.getAll(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return resource, nil
}

func (s *Service) getAll(ctx context.Context, options GetOptions) (Resource, error) {
	catalogCollection := &Collection{}
	var err error

	{
		catalogs := &applicationv1alpha1.CatalogList{}
		{
			err = s.client.List(ctx, catalogs, client.InNamespace(options.Namespace))
			if apimeta.IsNoMatchError(err) {
				return nil, microerror.Mask(noMatchError)
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else if len(catalogs.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, catalog := range catalogs.Items {
			// We hide catalog CRs from the giantswarm namespace by
			// default as these are internal.
			if options.AllNamespaces && catalog.Namespace == "giantswarm" {
				// Fall through
				continue
			}

			a := Catalog{
				CR: omitManagedFields(catalog.DeepCopy()),
			}
			catalogCollection.Items = append(catalogCollection.Items, a)
		}
	}

	return catalogCollection, nil
}

func (s *Service) getByName(ctx context.Context, namespace, name string, labelSelector labels.Selector) (Resource, error) {
	var err error

	var namespaces []string
	{
		if namespace != "" {
			// If the app CR has a catalog namespace we only check that.
			namespaces = []string{namespace}
		} else {
			// Otherwise we check the default namespace for public
			// catalogs and giantswarm for internal catalogs.
			namespaces = []string{metav1.NamespaceDefault, "giantswarm"}
		}
	}

	catalogCR := &applicationv1alpha1.Catalog{}

	for _, ns := range namespaces {
		err = s.client.Get(ctx, client.ObjectKey{
			Namespace: ns,
			Name:      name,
		}, catalogCR)
		if apierrors.IsNotFound(err) {
			// no-op
			continue
		} else if apimeta.IsNoMatchError(err) {
			return nil, microerror.Mask(noMatchError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	if catalogCR == nil || catalogCR.Name == "" {
		return nil, microerror.Maskf(notFoundError, "catalog %#q", name)
	}

	entries := &applicationv1alpha1.AppCatalogEntryList{}
	{
		lo := &client.ListOptions{
			LabelSelector: labelSelector,
		}
		err = s.client.List(ctx, entries, lo)
		if apimeta.IsNoMatchError(err) {
			return nil, microerror.Mask(noMatchError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(entries.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	catalog := &Catalog{
		CR:      omitManagedFields(catalogCR.DeepCopy()),
		Entries: entries,
	}
	catalog.CR.TypeMeta = metav1.TypeMeta{
		APIVersion: "catalog.application.giantswarm.io/v1alpha1",
		Kind:       "Catalog",
	}

	return catalog, nil
}

func (s *Service) ListCatalogEntries(ctx context.Context, options ListOptions) (Resource, error) {
	var catalogEntryList = applicationv1alpha1.AppCatalogEntryList{}
	{

		o := &client.ListOptions{
			LabelSelector: options.LabelSelector,
		}
		err := s.client.List(ctx, &catalogEntryList, o)
		if apimeta.IsNoMatchError(err) {
			return nil, microerror.Mask(noMatchError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else if len(catalogEntryList.Items) == 0 {
			return nil, microerror.Mask(noResourcesError)
		}
	}

	c := &CatalogEntryList{
		catalogEntryList,
	}
	return c, nil
}

// omitManagedFields removes managed fields to make YAML output easier to read.
// With Kubernetes 1.21 we can use OmitManagedFieldsPrinter and remove this.
func omitManagedFields(catalog *applicationv1alpha1.Catalog) *applicationv1alpha1.Catalog {
	catalog.ManagedFields = nil
	return catalog
}
