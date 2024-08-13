package base

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
)

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed cluster_config.yaml.tmpl
var clusterConfig string

func GetClusterBaseTemplates() []common.Template {
	return []common.Template{
		{Name: "kustomization.yaml", Data: kustomization},
		{Name: "cluster.yaml", Data: cluster},
		{Name: "cluster_config.yaml", Data: clusterConfig},
	}
}
