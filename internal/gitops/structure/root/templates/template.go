package root

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
)

//go:embed sops.yaml.tmpl
var sops string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetRepositoryRootTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: ".sops.yaml", Data: sops},
	}
}
