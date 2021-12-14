package openstack

import (
	_ "embed"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed cluster_class.yaml.tmpl
var clusterClass string

//go:embed open_stack_cluster_template.yaml.tmpl
var openStackCluster string

//go:embed kubeadm_control_plane_template.yaml.tmpl
var kubeadmControlPlane string

//go:embed kubeadm_config_template.yaml.tmpl
var kubeadmConfigTemplate string

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
		{Name: "cluster_class.yaml.tmpl", Data: clusterClass},
		{Name: "cluster.yaml.tmpl", Data: cluster},
		{Name: "open_stack_cluster_template.yaml.tmpl", Data: openStackCluster},
		{Name: "kubeadm_control_plane_template.yaml.tmpl", Data: kubeadmControlPlane},
		{Name: "kubeadm_config_template.yaml.tmpl", Data: kubeadmConfigTemplate},
		{Name: "open_stack_machine_template.yaml.tmpl", Data: openStackMachineTemplate},
	}
}
