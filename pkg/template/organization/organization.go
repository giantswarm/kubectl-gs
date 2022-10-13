package organization

import (
	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Name string
}

func NewOrganizationCR(config Config) (*securityv1alpha1.Organization, error) {
	orgCR := &securityv1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Organization",
			APIVersion: "security.giantswarm.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
		},
		Spec: securityv1alpha1.OrganizationSpec{},
	}

	return orgCR, nil
}
