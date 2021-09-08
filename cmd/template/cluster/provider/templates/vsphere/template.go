package vsphere

import (
	_ "embed"
	"strings"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed vsphere_cluster.yaml.tmpl
var vsphereCluster string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed vsphere_machine_template.yaml.tmpl
var vsphereMachineTemplate string

// GetTemplate merges all .tmpl files.
func GetTemplate() string {
	return strings.Join([]string{
		cluster,
		vsphereCluster,
		kubeadmControlPlane,
		vsphereMachineTemplate,
		// Adds a separator at the end of the joined template for easier joinability of cluster and node pool templates.
		"",
	}, "\n---\n")
}
