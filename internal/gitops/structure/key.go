package structure

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	fluxKustomizationAPIVersion = "kustomize.toolkit.fluxcd.io/v1beta2"
	fluxKustomizationKind       = "Kustomization"

	organizationAPIVersion = "security.giantswarm.io/v1alpha1"
	organizationKind       = "Organization"

	directoryManagementClusters = "management-clusters"
	directoryOrganizations      = "organizations"
	directorySecrets            = "secrets"
	directorySOPSPublicKeys     = ".sops.keys"
	directoryWorkloadClusters   = "workload-clusters"

	mainKustomizationSuffix     = "gitops"
	clustersKustomizationPrefix = "clusters"

	kustomizationsNamespace  = "default"
	kustomizationsSourceKind = "GitRepository"
)

func fileName(name string) string {
	return fmt.Sprintf("%s.yaml", name)
}

func kustomizationName(mc, wc string) string {
	if wc == "" {
		return fmt.Sprintf("%s-%s", mc, mainKustomizationSuffix)
	} else {
		return fmt.Sprintf("%s-%s-%s", mc, clustersKustomizationPrefix, wc)
	}
}

func kustomizationPath(mc, org, wc string) string {
	if wc == "" || org == "" {
		return fmt.Sprintf("./%s/%s", directoryManagementClusters, mc)
	} else {
		return fmt.Sprintf(
			"./%s/%s/%s/%s/%s/%s",
			directoryManagementClusters,
			mc,
			directoryOrganizations,
			org,
			directoryWorkloadClusters,
			wc,
		)
	}
}

func kustomizationManifest(name, path, repository, serviceAccount, interval, timeout string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fluxKustomizationAPIVersion,
			"kind":       fluxKustomizationKind,
			"metadata": map[string]string{
				"name":      name,
				"namespace": kustomizationsNamespace,
			},
			"spec": map[string]interface{}{
				"interval":           interval,
				"path":               path,
				"prune":              false,
				"serviceAccountName": serviceAccount,
				"sourceRef": map[string]string{
					"kind": kustomizationsSourceKind,
					"name": repository,
				},
				"timeout": timeout,
			},
		},
	}
}

func organizationManifest(name string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": organizationAPIVersion,
			"kind":       organizationKind,
			"metadata": map[string]string{
				"name": name,
			},
			"spec": map[string]interface{}{},
		},
	}
}
