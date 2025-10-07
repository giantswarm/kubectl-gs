package workcluster

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
)

//go:embed apps_kustomization.yaml.tmpl
var appsKustomization string

//go:embed cluster_userconfig.yaml.tmpl
var clusterUserConfig string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed patch_cluster_config.yaml.tmpl
var patchClusterConfig string

//go:embed patch_cluster_userconfig.yaml.tmpl
var patchClusterUserconfig string

//go:embed private-key.yaml.tmpl
var privateKey string

//go:embed workload-cluster.yaml.tmpl
var workloadCluster string

func GetAppsDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "kustomization.yaml", Data: appsKustomization},
		common.Template{Name: "patch_cluster_config.yaml", Data: patchClusterConfig},
	}
}

func GetClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "kustomization.yaml", Data: kustomization},
		common.Template{Name: "cluster_userconfig.yaml", Data: clusterUserConfig},
		common.Template{Name: "patch_cluster_userconfig.yaml", Data: patchClusterUserconfig},
	}
}

func GetSecretsDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .WorkloadCluster }}.gpgkey.enc.yaml", Data: privateKey},
	}
}

func GetWorkloadClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .WorkloadCluster }}.yaml", Data: workloadCluster},
	}
}
