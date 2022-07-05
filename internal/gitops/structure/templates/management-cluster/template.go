package organization

import (
	_ "embed"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

// GetManagementClusterTemplate merges .tmpl files for organization layer.
func GetManagementClusterTemplates() []string {
	return []string{
		managementCluster,
	}
}
