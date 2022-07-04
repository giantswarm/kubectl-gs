package organization

import (
	_ "embed"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

// GetManagementClusterTemplate merges .tmpl files for organization layer.
func GetManagementClusterTemplate() []string {
	return []string{
		managementCluster,
	}
}
