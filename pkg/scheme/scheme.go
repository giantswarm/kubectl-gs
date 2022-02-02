package scheme

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func NewScheme() (*runtime.Scheme, error) {
	schemeBuilder := runtime.NewSchemeBuilder(
		capiv1alpha3.AddToScheme,
		infrastructurev1alpha3.AddToScheme,
		applicationv1alpha1.AddToScheme,
	)
	scheme := runtime.NewScheme()
	err := schemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return scheme, nil
}
