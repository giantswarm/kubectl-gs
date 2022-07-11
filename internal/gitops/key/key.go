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
)

func FileName(name string) string {
	return fmt.Sprintf("%s.yaml", name)
}

func OrganizationsDirectory(mc string) string {
	return fmt.Sprintf("%s/%s/%s", DirectoryManagementClusters, mc, DirectoryOrganizations)
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
