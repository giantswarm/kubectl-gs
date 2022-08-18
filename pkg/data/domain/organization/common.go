package organization

import (
	"context"
	"sort"

	securityv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func (s *Service) getByName(ctx context.Context, name string) (Resource, error) {
	var err error

	o := &Organization{}
	{
		org := &securityv1alpha1.Organization{}
		err = s.client.CtrlClient().Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: metav1.NamespaceNone,
		}, org)

		if apierrors.IsNotFound(err) {
			return nil, microerror.Mask(notFoundError)
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		o.Organization = omitManagedFields(org)
		o.Organization.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   securityv1alpha1.SchemeGroupVersion.Group,
			Version: securityv1alpha1.SchemeGroupVersion.Version,
			Kind:    "Organization",
		})
	}

	return o, nil
}

func (s *Service) getAll(ctx context.Context) (Resource, error) {
	accessReview, err := s.getSelfSubjectAccessReview(ctx)

	if err != nil {
		return nil, microerror.Mask(err)
	} else if accessReview.Status.Allowed {
		return s.getAllCRs(ctx)
	} else {
		return s.getAllPermittedOrgResources(ctx)
	}
}

func (s *Service) getAllCRs(ctx context.Context) (Resource, error) {
	var err error
	orgCollection := &Collection{}
	{
		orgs := &securityv1alpha1.OrganizationList{}
		{
			err = s.client.CtrlClient().List(ctx, orgs)
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

func (s *Service) getAllPermittedOrgResources(ctx context.Context) (Resource, error) {
	rulesReview, err := s.getSelfSubjectRulesReview(ctx)

	if err != nil {
		return nil, microerror.Mask(err)
	}

	contains := func(slice []string, value string) bool {
		for _, item := range slice {
			if item == value {
				return true
			}
		}
		return false
	}

	var orgNames []string
	for _, rule := range rulesReview.Status.ResourceRules {
		if contains(rule.Verbs, "get") && contains(rule.Resources, "organizations") {
			orgNames = append(orgNames, rule.ResourceNames...)
		}
	}

	if len(orgNames) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	sort.Strings(orgNames)

	return s.getByNames(ctx, orgNames)
}

func (s *Service) getByNames(ctx context.Context, names []string) (*Collection, error) {

	orgCollection := &Collection{}

	for _, name := range names {
		resource, err := s.getByName(ctx, name)

		if err != nil {
			return nil, microerror.Mask(err)
		}

		org := resource.(*Organization)
		if org == nil {
			continue
		}

		o := Organization{
			Organization: org.Organization.DeepCopy(),
		}

		orgCollection.Items = append(orgCollection.Items, o)
	}

	if len(orgCollection.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	return orgCollection, nil
}

// omitManagedFields removes managed fields to make YAML output easier to read.
// With Kubernetes 1.21 we can use OmitManagedFieldsPrinter and remove this.
func omitManagedFields(org *securityv1alpha1.Organization) *securityv1alpha1.Organization {
	org.ManagedFields = nil
	return org
}
