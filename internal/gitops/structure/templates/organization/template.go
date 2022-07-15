package organization

import (
	_ "embed"
)

//go:embed organization.yaml.tmpl
var organization string

//go:embed kustomization.yaml.tmpl
var kustomization string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetOrganizationDirectoryTemplates() map[string]string {
	return map[string]string{
		"{{ .Name }}.yaml": organization,
	}
}

func GetWorkloadClustersDirectoryTemplates() map[string]string {
	return map[string]string{
		"kustomization.yaml": kustomization,
	}
}
