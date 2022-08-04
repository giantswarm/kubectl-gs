package workcluster

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/common"
)

//go:embed apps_kustomization.yaml.tmpl
var appsKustomization string

//go:embed cluster_userconfig.yaml.tmpl
var clusterUserConfig string

//go:embed default_apps_userconfig.yaml.tmpl
var defaultAppsUserConfig string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed patch_cluster_config.yaml.tmpl
var patchClusterConfig string

//go:embed patch_cluster_userconfig.yaml.tmpl
var patchClusterUserconfig string

//go:embed patch_default_apps_userconfig.yaml.tmpl
var patchDefaultAppsUserconfig string

//go:embed workload-cluster.yaml.tmpl
var workloadCluster string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetWorkloadClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .Name }}.yaml", Data: workloadCluster},
	}
}

func GetClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "kustomization.yaml", Data: kustomization},
		common.Template{Name: "cluster_userconfig.yaml", Data: clusterUserConfig},
		common.Template{Name: "default_apps_userconfig.yaml", Data: defaultAppsUserConfig},
		common.Template{Name: "patch_cluster_userconfig.yaml", Data: patchClusterUserconfig},
		common.Template{Name: "patch_default_apps_userconfig.yaml", Data: patchDefaultAppsUserconfig},
	}
}

func GetAppsDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "kustomization.yaml", Data: appsKustomization},
		common.Template{Name: "patch_cluster_config.yaml", Data: patchClusterConfig},
	}
}
