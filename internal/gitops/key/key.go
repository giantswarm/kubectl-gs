package key

import (
	"fmt"
)

const (
	DirectoryClusterApps        = "apps"
	DirectoryClusterDefinition  = "cluster"
	DirectoryManagementClusters = "management-clusters"
	DirectoryOrganizations      = "organizations"
	DirectorySecrets            = "secrets"
	DirectorySOPSPublicKeys     = ".sops.keys"
	DirectoryWorkloadClusters   = "workload-clusters"

	FileKustomization = "kustomization.yaml"
)

// TBD: Better naming of the functions and explanations, right now it may be confusing,
// because names are sometimes very similar, also paths themselves are relative what may
// be extra confusing.
func GetAppsKustomization() string {
	return fmt.Sprintf("%s/%s", DirectoryClusterApps, FileKustomization)
}

func GetOrgDir(path string) string {
	return fmt.Sprintf("%s/%s", path, DirectoryOrganizations)
}

func GetSecretsDir(path string) string {
	return fmt.Sprintf("%s/%s", path, DirectorySecrets)
}

func GetSopsDir(path string) string {
	return fmt.Sprintf("%s/%s", path, DirectorySOPSPublicKeys)
}

func GetWCDir(name string) string {
	return fmt.Sprintf("%s/%s", DirectoryWorkloadClusters, name)
}

func GetWCAppDir(name string) string {
	return fmt.Sprintf("%s/%s", DirectoryClusterApps, name)
}

func GetWCAppsDir(name string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryWorkloadClusters, name, DirectoryClusterApps)
}

func GetWCClusterDir(name string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryWorkloadClusters, name, DirectoryClusterDefinition)
}

func GetWCsDir(path string) string {
	return fmt.Sprintf("%s/%s", path, DirectoryWorkloadClusters)
}

func GetWCsKustomization() string {
	return fmt.Sprintf("%s/%s", DirectoryWorkloadClusters, FileKustomization)
}

func FileName(name string) string {
	return fmt.Sprintf("%s.yaml", name)
}

func OrganizationsDirectory(mc string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryManagementClusters, mc, DirectoryOrganizations)
}

func WorkloadClusterAppDirectory(mc, org, wc string) string {
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

func WorkloadClustersDirectory(mc, org string) string {
	return fmt.Sprintf(
		"%s/%s/%s/%s",
		DirectoryManagementClusters,
		mc,
		DirectoryOrganizations,
		org,
	)
}
