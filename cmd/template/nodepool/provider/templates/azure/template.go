package azure

import (
	_ "embed"
	"strings"
)

//go:embed machine_pool.yaml.tmpl
var machinePool string

//go:embed azure_machine_pool.yaml.tmpl
var azureMachinePool string

//go:embed kubeadm_config.yaml.tmpl
var kubeadmConfig string

// GetTemplate merges all .tmpl files.
func GetTemplate() string {
	return strings.Join([]string{
		machinePool,
		azureMachinePool,
		kubeadmConfig,
		// Adds a separator at the end of the joined template for easier joinability of cluster and node pool templates.
		"",
	}, "\n---\n")
}
