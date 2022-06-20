package structure

import (
	"fmt"
)

const (
	fluxKustomizationAPIVersion = "kustomize.toolkit.fluxcd.io/v1beta2"
	fluxKustomizationKind       = "Kustomization"
	fluxSourceKind              = "GitRepository"

	topLevelGitOpsDirectory     = "management-clusters"
	topLevelKustomizationSuffix = "gitops"
	clustersKustomizationPrefix = "clusters"

	kustomizationNamespace = "default"

	secretsDirectory          = "secrets"
	sopsPublicKeysDirectory   = ".sops.keys"
	organizationsDirectory    = "organizations"
	workloadClustersDirectory = "workload-clusters"
)

func kustomizationFileName(name string) string {
	return fmt.Sprintf("%s.yaml", name)
}

func kustomizationName(mc, wc string) string {
	if wc == "" {
		return fmt.Sprintf("%s-%s", mc, topLevelKustomizationSuffix)
	} else {
		return fmt.Sprintf("%s-%s-%s", mc, clustersKustomizationPrefix, wc)
	}
}

func kustomizationPath(mc, org, wc string) string {
	if wc == "" || org == "" {
		return fmt.Sprintf("./%s/%s", topLevelGitOpsDirectory, mc)
	} else {
		return fmt.Sprintf("./%s/%s/%s/%s/%s/%s", topLevelGitOpsDirectory, mc, organizationsDirectory, org, workloadClustersDirectory, wc)
	}
}
