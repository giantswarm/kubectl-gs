package release

import (
	"context"

	"github.com/giantswarm/microerror"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByName(ctx, options.Name, options.Namespace, options.ActiveOnly)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAll(ctx, options.Namespace, options.ActiveOnly)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}

func (s *Service) getByName(ctx context.Context, name, namespace string, activeOnly bool) (Resource, error) {
	var err error

	r := &Release{}
	{
		rel := &releasev1alpha1.Release{}
		err = s.client.Get(ctx, runtimeclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, rel)
		if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if activeOnly && !rel.Status.Ready {
			return nil, microerror.Mask(noMatchError)
		}

		r.CR = omitManagedFields(rel)
		r.CR.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   releasev1alpha1.SchemeGroupVersion.Group,
			Version: releasev1alpha1.SchemeGroupVersion.Version,
			Kind:    releasev1alpha1.NewReleaseCR().Kind,
		})
	}

	return r, nil
}

func (s *Service) getAll(ctx context.Context, namespace string, activeOnly bool) (Resource, error) {
	var err error

	collection := &ReleaseCollection{}
	{
		releases := &releasev1alpha1.ReleaseList{}
		{
			err = s.client.List(ctx, releases, &runtimeclient.ListOptions{
				Namespace: namespace,
			})
			if apimeta.IsNoMatchError(err) {
				return nil, microerror.Mask(noMatchError)
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else if len(releases.Items) == 0 {
				return nil, microerror.Mask(noResourcesError)
			}
		}

		for _, rel := range releases.Items {
			if activeOnly && !rel.Status.Ready {
				continue
			}

			r := Release{
				CR: omitManagedFields(rel.DeepCopy()),
			}
			collection.Items = append(collection.Items, r)
		}
	}

	return collection, nil
}

// omitManagedFields removes managed fields to make YAML output easier to read.
// With Kubernetes 1.21 we can use OmitManagedFieldsPrinter and remove this.
func omitManagedFields(rel *releasev1alpha1.Release) *releasev1alpha1.Release {
	rel.ManagedFields = nil
	return rel
}
