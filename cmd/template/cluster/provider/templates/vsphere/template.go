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

// GetTemplate merges all .tmpl files.
func GetTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		cluster,
		vsphereCluster,
		kubeadmControlPlane,
		kubeadmConfigTemplate,
		machineDeployment,
		vsphereMachineTemplate,
	}
}
