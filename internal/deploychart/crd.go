package deploychart

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OCIRepositoryCRDName = "ocirepositories.source.toolkit.fluxcd.io"
	HelmReleaseCRDName   = "helmreleases.helm.toolkit.fluxcd.io"
)

// FluxCRDVersions holds the detected API versions for the Flux CRDs.
type FluxCRDVersions struct {
	OCIRepositoryAPIVersion string // e.g. "source.toolkit.fluxcd.io/v1"
	HelmReleaseAPIVersion   string // e.g. "helm.toolkit.fluxcd.io/v2"
}

// DetectFluxCRDVersions queries the cluster for the available CRD versions
// of OCIRepository and HelmRelease. It returns the storage version for each.
func DetectFluxCRDVersions(ctx context.Context, extClient apiextensionsclientset.Interface) (FluxCRDVersions, error) {
	var result FluxCRDVersions

	ociCRD, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, OCIRepositoryCRDName, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("flux CRD %q not found in cluster — ensure Flux is installed: %w", OCIRepositoryCRDName, err)
	}

	hrCRD, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, HelmReleaseCRDName, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("flux CRD %q not found in cluster — ensure Flux is installed: %w", HelmReleaseCRDName, err)
	}

	result.OCIRepositoryAPIVersion, err = storageAPIVersion(ociCRD.Spec.Group, ociCRD.Spec.Versions)
	if err != nil {
		return result, fmt.Errorf("determining API version for %s: %w", OCIRepositoryCRDName, err)
	}

	result.HelmReleaseAPIVersion, err = storageAPIVersion(hrCRD.Spec.Group, hrCRD.Spec.Versions)
	if err != nil {
		return result, fmt.Errorf("determining API version for %s: %w", HelmReleaseCRDName, err)
	}

	return result, nil
}

// storageAPIVersion finds the storage version from a CRD's version list
// and returns the full API version string (group/version).
func storageAPIVersion(group string, versions []apiextensionsv1.CustomResourceDefinitionVersion) (string, error) {
	for _, v := range versions {
		if v.Storage {
			return group + "/" + v.Name, nil
		}
	}
	// Fallback: first served version.
	for _, v := range versions {
		if v.Served {
			return group + "/" + v.Name, nil
		}
	}
	return "", fmt.Errorf("no served version found")
}
