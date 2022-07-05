package organization

import (
	_ "embed"
)

//go:embed organization.yaml.tmpl
var organization string

// OrganizationDirectoryTemplate returns organization directory layout.
func OrganizationDirectoryTemplates() []string {
	return []string{
		organization,
	}
}
