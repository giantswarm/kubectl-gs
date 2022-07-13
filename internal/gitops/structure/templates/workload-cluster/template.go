package organization

import (
	_ "embed"
)

//go:embed cluster_userconfig.yaml.tmpl
var clusterUserConfig string

//go:embed default_apps_userconfig.yaml.tmpl
var defaultAppsUserConfig string

//go:embed cluster_kustomization.yaml.tmpl
var clusterKustomization string

//go:embed patch_cluster_userconfig.yaml.tmpl
var patchClusterUserconfig string

//go:embed patch_default_apps_userconfig.yaml.tmpl
var patchDefaultAppsUserconfig string

//go:embed wcs_kustomization.yaml.tmpl
var worklodClustersKustomization string

//go:embed workload-cluster.yaml.tmpl
var workloadCluster string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetWorkloadClusterDirectoryTemplates() map[string]string {
	return map[string]string{
		"{{ .Name }}":   workloadCluster,
		"kustomization": worklodClustersKustomization,
	}
}

func GetClusterDirectoryTemplates() map[string]string {
	return map[string]string{
		"cluster_userconfig":            clusterUserConfig,
		"default_apps_userconfig":       defaultAppsUserConfig,
		"kustomization":                 clusterKustomization,
		"patch_cluster_userconfig":      patchClusterUserconfig,
		"patch_default_apps_userconfig": patchDefaultAppsUserconfig,
	}
}
