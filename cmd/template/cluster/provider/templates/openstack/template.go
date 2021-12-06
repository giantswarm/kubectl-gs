package openstack

import (
	_ "embed"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed open_stack_cluster.yaml.tmpl
var openStackCluster string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed kubeadm_config_template.yaml.tmpl
var kubeadmConfigTemplate string

//go:embed machine_deployment.yaml.tmpl
var machineDeployment string

//go:embed open_stack_machine_template.yaml.tmpl
var openStackMachineTemplate string

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
		{Name: "open_stack_cluster.yaml.tmpl", Data: openStackCluster},
		{Name: "kubeadm_control_plane.yaml.tmpl", Data: kubeadmControlPlane},
		{Name: "kubeadm_config_template.yaml.tmpl", Data: kubeadmConfigTemplate},
		{Name: "machine_deployment.yaml.tmpl", Data: machineDeployment},
		{Name: "open_stack_machine_template.yaml.tmpl", Data: openStackMachineTemplate},
	}
}
