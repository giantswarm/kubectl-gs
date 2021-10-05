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

type Template struct {
	Name string
	Data string
}

// GetTemplate merges all .tmpl files.
func GetTemplates() []Template {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []Template{
		{Name: "cluster.yaml.tmpl", Data: cluster},
		{Name: "azure_cluster.yaml.tmpl", Data: azureCluster},
		{Name: "kubeadm_control_plane.yaml.tmpl", Data: kubeadmControlPlane},
		{Name: "azure_machine_template.yaml.tmpl", Data: azureMachineTemplate},
	}
}
