package key

import commonkey "github.com/giantswarm/kubectl-gs/v6/internal/key"

// ClusterBaseProviders returns the providers `gitops add base` writes a
// cluster base for, in `bases/clusters/PROVIDER`.
func ClusterBaseProviders() []string {
	return []string{
		commonkey.ProviderCAPA,
		commonkey.ProviderCAPZ,
		commonkey.ProviderVSphere,
	}
}
