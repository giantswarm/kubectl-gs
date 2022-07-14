package organization

import (
	_ "embed"
)

//go:embed cluster_userconfig.yaml.tmpl
var clusterUserConfig string

//go:embed default_apps_userconfig.yaml.tmpl
var defaultAppsUserConfig string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed patch_cluster_userconfig.yaml.tmpl
var patchClusterUserconfig string

//go:embed patch_default_apps_userconfig.yaml.tmpl
var patchDefaultAppsUserconfig string

//go:embed workload-cluster.yaml.tmpl
var workloadCluster string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetWorkloadClusterDirectoryTemplates() map[string]string {
	return map[string]string{
		"{{ .Name }}": workloadCluster,
	}
}

func GetClusterDirectoryTemplates() map[string]string {
	return map[string]string{
		"cluster_userconfig":            clusterUserConfig,
		"default_apps_userconfig":       defaultAppsUserConfig,
		"kustomization":                 kustomization,
		"patch_cluster_userconfig":      patchClusterUserconfig,
		"patch_default_apps_userconfig": patchDefaultAppsUserconfig,
	}
}
