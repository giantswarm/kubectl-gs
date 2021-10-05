package vsphere

import (
	_ "embed"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed vsphere_cluster.yaml.tmpl
var vsphereCluster string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed kubeadm_config_template.yaml.tmpl
var kubeadmConfigTemplate string

//go:embed machine_deployment.yaml.tmpl
var machineDeployment string

//go:embed vsphere_machine_template.yaml.tmpl
var vsphereMachineTemplate string

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
		{Name: "vsphere_cluster.yaml.tmpl", Data: vsphereCluster},
		{Name: "kubeadm_control_plane.yaml.tmpl", Data: kubeadmControlPlane},
		{Name: "kubeadm_config_template.yaml.tmpl", Data: kubeadmConfigTemplate},
		{Name: "machine_deployment.yaml.tmpl", Data: machineDeployment},
		{Name: "vsphere_machine_template.yaml.tmpl", Data: vsphereMachineTemplate},
	}
}
