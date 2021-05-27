package catalog

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid catalog getter Service.
type Config struct {
	Client *client.Client
}

// Service is the object we'll hang the catalog getter methods on.
type Service struct {
	client *client.Client
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

	resource, err = s.getAll(ctx, options.LabelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return resource, nil
}

func (s *Service) getAll(ctx context.Context, labelSelector labels.Selector) (Resource, error) {
	catalogCollection := &Collection{}
	var err error

	{
		lo := &runtimeclient.ListOptions{
			LabelSelector: labelSelector,
		}

		catalogs := &applicationv1alpha1.CatalogList{}
		{
			err = s.client.K8sClient.CtrlClient().List(ctx, catalogs, lo)
			if apimeta.IsNoMatchError(err) {
				return nil, microerror.Mask(noMatchError)
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else if len(catalogs.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, catalog := range catalogs.Items {
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

	catalogCR := &applicationv1alpha1.Catalog{}
	{
		err = s.client.K8sClient.CtrlClient().Get(ctx, runtimeclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, catalogCR)
		if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if apimeta.IsNoMatchError(err) {
			return nil, microerror.Mask(noMatchError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	entries := &applicationv1alpha1.AppCatalogEntryList{}
	{
		lo := &runtimeclient.ListOptions{
			LabelSelector: labelSelector,
		}
		err = s.client.K8sClient.CtrlClient().List(ctx, entries, lo)
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

// omitManagedFields removes managed fields to make YAML output easier to read.
// With Kubernetes 1.21 we can use OmitManagedFieldsPrinter and remove this.
func omitManagedFields(catalog *applicationv1alpha1.Catalog) *applicationv1alpha1.Catalog {
	catalog.ManagedFields = nil
	return catalog
}
