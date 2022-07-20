package organization

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/common"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetManagementClusterTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .Name }}.yaml", Data: managementCluster},
	}
}
