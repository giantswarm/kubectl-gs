package organization

import (
	_ "embed"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetManagementClusterTemplates() map[string]string {
	return map[string]string{
		"{{ .Name }}": managementCluster,
	}
}
