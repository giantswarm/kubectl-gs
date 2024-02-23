package scheme

import (
	application "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	gscore "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	infrastructure "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	provider "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	release "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	k8score "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capainfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta1"
	eks "sigs.k8s.io/cluster-api-provider-aws/v2/controlplane/eks/api/v1beta2"
	capaexp "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

func NewSchemeBuilder() []func(*runtime.Scheme) error {
	return []func(*runtime.Scheme) error{
		apiextensions.AddToScheme,    // CustomResourceDefinition
		application.AddToScheme,      // App, Catalog
		capi.AddToScheme,             // Cluster
		capiexp.AddToScheme,          // MachinePool
		capaexp.AddToScheme,          // AWSMachinePool
		eks.AddToScheme,              // EKS CRs
		k8score.AddToScheme,          // Secret, ConfigMap
		infrastructure.AddToScheme,   // AWSCluster (Giant Swarm CAPI)
		capz.AddToScheme,             // AzureCluster
		capzexp.AddToScheme,          // AzureMachinePool
		gscore.AddToScheme,           // Spark
		provider.AddToScheme,         // AWSConfig/AzureConfig
		release.AddToScheme,          // Release
		securityv1alpha1.AddToScheme, // Organizations
		capainfrav1.AddToScheme,      // AWSCluster (CAPA)
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
