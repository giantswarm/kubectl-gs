package key

import "fmt"

const (
	namespaceDirectory       = "namespace"
	namespaceFile            = "namespace.yaml"
	workloadClusterKeysRegex = "%s/organizations/[^/]+/namespace/secrets/.*\\.(gpg|age)key\\.enc\\.yaml"
)

// NamespaceDirName is the directory of what an organisation keeps in its
// namespace, which the organisation's namespace Kustomization applies.
func NamespaceDirName() string {
	return namespaceDirectory
}

// NamespaceFileName is the file of an organisation's namespace Kustomization,
// next to the `namespace` directory it applies.
func NamespaceFileName() string {
	return namespaceFile
}

// WorkloadClusterKeysRegex matches the workload clusters' SOPS keys in the
// organisations' `namespace/secrets` under the management cluster directory
// mcDir. The master key encrypts them, as the organisations' namespace
// Kustomizations decrypt them with it.
func WorkloadClusterKeysRegex(mcDir string) string {
	return fmt.Sprintf(workloadClusterKeysRegex, mcDir)
}
