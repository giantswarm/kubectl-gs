package scheme

import (
	application "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	gscore "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	infrastructure "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	provider "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	release "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/microerror"
	k8score "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	expcapz "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	expcapi "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
)

func NewSchemeBuilder() []func(*runtime.Scheme) error {
	return []func(*runtime.Scheme) error{
		apiextensions.AddToScheme,    // CustomResourceDefinition
		application.AddToScheme,      // App, Catalog
		capi.AddToScheme,             // Cluster
		expcapi.AddToScheme,          // AWSMachinePool
		k8score.AddToScheme,          // Secret, ConfigMap
		infrastructure.AddToScheme,   // AWSCluster (Giant Swarm CAPI)
		capz.AddToScheme,             // AzureCluster
		expcapz.AddToScheme,          // AzureMachinePool
		gscore.AddToScheme,           // Spark
		provider.AddToScheme,         // AWSConfig/AzureConfig
		release.AddToScheme,          // Release
		securityv1alpha1.AddToScheme, // Organizations
		networking.AddToScheme,       // Ingress
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
