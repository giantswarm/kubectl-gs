package structure

const (
	fluxKustomizationAPIVersion = "kustomize.toolkit.fluxcd.io/v1beta2"
	fluxKustomizationKind       = "Kustomization"

	topLevelGitOpsDirectory     = "management-clusters"
	topLevelKustomizationSuffix = "gitops"

	secretsDirectory        = "secrets"
	sopsPublicKeysDirectory = ".sops.keys"
	organizationsDirectory  = "organizations"
)
