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

//go:embed bastion_secret.yaml.tmpl
var bastionSecret string

//go:embed bastion_machine_deployment.yaml.tmpl
var bastionMachineDeployment string

//go:embed bastion_azure_machine_template.yaml.tmpl
var bastionAzureMachineTemplate string

// GetTemplates return a list of templates.
func GetTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		cluster,
		azureCluster,
		kubeadmControlPlane,
		azureMachineTemplate,
		bastionSecret,
		bastionMachineDeployment,
		bastionAzureMachineTemplate,
	}
}
