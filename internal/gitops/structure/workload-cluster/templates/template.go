package workcluster

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

//go:embed apps_kustomization.yaml.tmpl
var appsKustomization string

//go:embed cluster_userconfig.yaml.tmpl
var clusterUserConfig string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed patch_cluster_userconfig.yaml.tmpl
var patchClusterUserconfig string

//go:embed workload-cluster.yaml.tmpl
var workloadCluster string

//go:embed workload-cluster-apps.yaml.tmpl
var workloadClusterApps string

//go:embed wc_kustomization.yaml.tmpl
var wcKustomization string

func GetAppsDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: key.SigsKustomizationFileName(), Data: appsKustomization},
	}
}

func GetClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: key.SigsKustomizationFileName(), Data: kustomization},
		common.Template{Name: "cluster_userconfig.yaml", Data: clusterUserConfig},
		common.Template{Name: "patch_cluster_userconfig.yaml", Data: patchClusterUserconfig},
	}
}

// GetWorkloadClustersDirectoryTemplates returns the cluster's Flux
// Kustomizations, which live next to the cluster's directory.
func GetWorkloadClustersDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .WorkloadCluster }}.yaml", Data: workloadCluster},
		common.Template{Name: "{{ .WorkloadCluster }}-apps.yaml", Data: workloadClusterApps},
	}
}

// GetWorkloadClusterDirectoryTemplates returns the files at the root of the
// cluster's directory.
func GetWorkloadClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: key.SigsKustomizationFileName(), Data: wcKustomization},
	}
}
