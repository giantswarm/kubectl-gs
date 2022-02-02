package scheme

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func NewSchemeBuilder() []func(*runtime.Scheme) error {
	return []func(*runtime.Scheme) error{
		apiextensionsv1.AddToScheme,        // Needed for CustomResourceDefinitions
		applicationv1alpha1.AddToScheme,    // Needed by app service for Apps and Catalogs
		capi.AddToScheme,                   // Needed by cluster service for CAPI Clusters
		corev1.AddToScheme,                 // Needed by app service for Secrets and ConfigMaps
		infrastructurev1alpha3.AddToScheme, // Needed by cluster service for pre-CAPI Clusters
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
