package organization

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
)

//go:embed organization.yaml.tmpl
var organization string

//go:embed kustomization.yaml.tmpl
var kustomization string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetOrganizationDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .Organization }}.yaml", Data: organization},
	}
}

func GetWorkloadClustersDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "kustomization.yaml", Data: kustomization},
	}
}
