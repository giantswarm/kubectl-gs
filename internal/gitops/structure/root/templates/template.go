package root

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
)

//go:embed pre-commit.tmpl
var preCommit string

//go:embed sops.yaml.tmpl
var sops string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetRepositoryRootTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: ".sops.yaml", Data: sops},
	}
}

func GetRepositoryGitTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "pre-commit", Data: preCommit, Permission: 0700},
	}
}
