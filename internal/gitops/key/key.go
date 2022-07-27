package key

import (
	"fmt"
)

const (
	DirectoryClusterApps        = "apps"
	DirectoryClusterDefinition  = "cluster"
	DirectoryAutomaticUpdates   = "automatic-updates"
	DirectoryManagementClusters = "management-clusters"
	DirectoryOrganizations      = "organizations"
	DirectorySecrets            = "secrets"
	DirectorySOPSPublicKeys     = ".sops.keys"
	DirectoryWorkloadClusters   = "workload-clusters"

	FileKustomization = "kustomization.yaml"
)

// Naming convention and explanation
//
// Since the `gitops/structure` package does not operate on absolute paths,
// the functions here must return paths relative to the layer currently
// being configure by the structure, e.g. Management Cluster. This however
// may be a bit confusing from the naming perspective because different
// commands must sometimes operate on the same directory, e.g. `add app` and
// `add image-update` both operate on the `apps` directory, but relatively seen
// slightly different, what is hard to capture in the function name. Hence the
// proposed naming convention is:
//
// Get + BASE + TARGET, where
// BASE - directory where we relatively starts building the path, not being part of the path.
// TARGET - directory or file we are targeting with the path.
//
// Examples:
// GetWcAppsKustomization - we start at `WC_NAME` and targets `apps/kustomization.yaml`
// GetMCsOrgDir - we start at `management-clusters` and targets `MC_NAME/organizations`
// GetAnySecretsDir - we start at any place and targets `ANY/secrets`

// GetWcAppsKustomization
// Base: WC_NAME
// Target: apps/kustomization.yaml
func GetWcAppsKustomizationFile() string {
	return fmt.Sprintf("%s/%s", DirectoryClusterApps, FileKustomization)
}

// GetAnySecretsDir
// Base: management-clusters
// Target: MC_NAME/organizations
func GetMCsOrgsDir(mc string) string {
	return fmt.Sprintf("%s/%s", mc, DirectoryOrganizations)
}

// GetAnySecretsDir
// Base: any
// Target: any/secrets
func GetAnySecretsDir(path string) string {
	return fmt.Sprintf("%s/%s", path, DirectorySecrets)
}

// GetMCsSopsDir
// Base: management-clusters
// Target: MC_NAME/.sops.keys
func GetMCsSopsDir(mc string) string {
	return fmt.Sprintf("%s/%s", mc, DirectorySOPSPublicKeys)
}

// GetOrgWcDir
// Base: ORG_NAME
// Target: workload-clusters/WC_NAME
func GetOrgWcDir(wc string) string {
	return fmt.Sprintf("%s/%s", DirectoryWorkloadClusters, wc)
}

// GetWcAppDir
// Base: WC_NAME
// Target: apps/APP_NAME
func GetWcAppDir(name string) string {
	return fmt.Sprintf("%s/%s", DirectoryClusterApps, name)
}

// GetOrgWcAppsDir
// Base: ORG_NAME
// Target: workload-clusters/WC_NAME/apps
func GetOrgWcAppsDir(name string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryWorkloadClusters, name, DirectoryClusterApps)
}

// GetOrgWcClusterDir
// Base: ORG_NAME
// Target: workload-clusters/WC_NAME/cluster
func GetOrgWcClusterDir(wc string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryWorkloadClusters, wc, DirectoryClusterDefinition)
}

// GetOrgWCsDir
// Base: organizations
// Target: ORG_NAME/workload-clusters
func GetOrgWCsDir(org string) string {
	return fmt.Sprintf("%s/%s", org, DirectoryWorkloadClusters)
}

// GetOrgWCsKustomizationFile
// Base: ORG_NAME
// Target: workload-clusters/kustomization.yaml
func GetOrgWCsKustomizationFile() string {
	return fmt.Sprintf("%s/%s", DirectoryWorkloadClusters, FileKustomization)
}

// GetRootOrganizationsDirectory
// Base: repository root
// Target: management-clusters/MC_NAME/organizations
func GetRootOrganizationsDirectory(mc string) string {
	return fmt.Sprintf("%s/%s/%s",
		DirectoryManagementClusters,
		mc,
		DirectoryOrganizations,
	)
}

// GetRootWorkloadClusterDirectory
// Base: repository root
// Target: management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
func GetRootWorkloadClusterDirectory(mc, org, wc string) string {
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s/%s",
		DirectoryManagementClusters,
		mc,
		DirectoryOrganizations,
		org,
		DirectoryWorkloadClusters,
		wc,
	)
}

// GetRootOrganizationDirectory
// Base: repository root
// Target: management-clusters/MC_NAME/organizations/ORG_NAME
func GetRootOrganizationDirectory(mc, org string) string {
	return fmt.Sprintf(
		"%s/%s/%s/%s",
		DirectoryManagementClusters,
		mc,
		DirectoryOrganizations,
		org,
	)
}
