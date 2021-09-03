package azure

import (
	_ "embed"
	"strings"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed azure_cluster.yaml.tmpl
var azureCluster string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed azure_machine_template.yaml.tmpl
var azureMachineTemplate string

// GetTemplate merges all .tmpl files.
func GetTemplate() string {
	return strings.Join([]string{
		cluster,
		azureCluster,
		kubeadmControlPlane,
		azureMachineTemplate,
		// Adds a separator at the end of the joined template for easier joinability of cluster and node pool templates.
		"",
	}, "\n---\n")
}
