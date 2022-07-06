package organization

import (
	_ "embed"
)

//go:embed organization.yaml.tmpl
var organization string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetOrganizationDirectoryTemplates() []string {
	return []string{
		organization,
	}
}
