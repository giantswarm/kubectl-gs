package base

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
)

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed cluster_config.yaml.tmpl
var clusterConfig string

//go:embed default_apps.yaml.tmpl
var defaultApps string

//go:embed default_apps_config.yaml
var defaultAppsConfig string

func GetClusterDirectoryTemplates() []common.Template {
	return []common.Template{
		{Name: "kustomization.yaml", Data: kustomization},
		{Name: "cluster.yaml", Data: cluster},
		{Name: "cluster_config.yaml", Data: clusterConfig},
		{Name: "default_apps.yaml", Data: defaultApps},
		{Name: "default_apps_config.yaml", Data: defaultAppsConfig},
	}
}
