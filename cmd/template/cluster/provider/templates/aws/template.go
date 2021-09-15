package aws

import (
	_ "embed"
)

//go:embed cluster.yaml.tmpl
var cluster string

//go:embed aws_cluster.yaml.tmpl
var awsCluster string

//go:embed kubeadm_control_plane.yaml.tmpl
var kubeadmControlPlane string

//go:embed aws_machine_template.yaml.tmpl
var awsMachineTemplate string

//go:embed aws_cluster_role_identity.yaml.tmpl
var awsClusterRoleIdentity string

//go:embed bastion_secret.yaml.tmpl
var bastionSecret string

//go:embed bastion_machine_deployment.yaml.tmpl
var bastionMachineDeployment string

//go:embed bastion_aws_machine_template.yaml.tmpl
var bastionAWSMachineTemplate string

// GetTemplate merges all .tmpl files.
func GetTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		cluster,
		awsCluster,
		kubeadmControlPlane,
		awsMachineTemplate,
		awsClusterRoleIdentity,
		bastionSecret,
		bastionMachineDeployment,
		bastionAWSMachineTemplate,
	}
}
