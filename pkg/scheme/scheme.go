package scheme

import (
	application "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	gscore "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	release "github.com/giantswarm/releases/sdk/api/v1alpha1"
	k8score "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewSchemeBuilder() []func(*runtime.Scheme) error {
	return []func(*runtime.Scheme) error{
		apiextensions.AddToScheme,    // CustomResourceDefinition
		application.AddToScheme,      // App, Catalog
		k8score.AddToScheme,          // Secret, ConfigMap
		gscore.AddToScheme,           // Spark
		release.AddToScheme,          // Release
		securityv1alpha1.AddToScheme, // Organizations
	}
}

func NewScheme() (*runtime.Scheme, error) {
	builder := runtime.NewSchemeBuilder(NewSchemeBuilder()...)
	scheme := runtime.NewScheme()
	err := builder.AddToScheme(scheme)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return scheme, nil
}
