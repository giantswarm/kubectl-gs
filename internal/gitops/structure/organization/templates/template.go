package organization

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

//go:embed organization.yaml.tmpl
var organization string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed org_kustomization.yaml.tmpl
var orgKustomization string

//go:embed namespace.yaml.tmpl
var namespace string

//go:embed namespace_kustomization.yaml.tmpl
var namespaceKustomization string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetOrganizationDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .Organization }}.yaml", Data: organization},
		common.Template{Name: key.SigsKustomizationFileName(), Data: orgKustomization},
		common.Template{Name: key.NamespaceFileName(), Data: namespace},
	}
}

// GetNamespaceDirectoryTemplates returns what the organisation keeps in its
// namespace, which namespace.yaml applies.
func GetNamespaceDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: key.SigsKustomizationFileName(), Data: namespaceKustomization},
	}
}

func GetWorkloadClustersDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: key.SigsKustomizationFileName(), Data: kustomization},
	}
}
