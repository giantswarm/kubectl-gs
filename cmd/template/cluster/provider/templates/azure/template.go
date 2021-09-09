package azure

import (
	_ "embed"
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
func GetTemplates() []string {
	return []string{
		cluster,
		azureCluster,
		kubeadmControlPlane,
		azureMachineTemplate,
	}
}
