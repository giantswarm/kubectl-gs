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
		{Name: "aws_cluster.yaml.tmpl", Data: awsCluster},
		{Name: "kubeadm_control_plane.yaml.tmpl", Data: kubeadmControlPlane},
		{Name: "aws_machine_template.yaml.tmpl", Data: awsMachineTemplate},
		{Name: "aws_cluster_role_identity.yaml.tmpl", Data: awsClusterRoleIdentity},
		{Name: "bastion_secret.yaml.tmpl", Data: bastionSecret},
		{Name: "bastion_machine_deployment.yaml.tmpl", Data: bastionMachineDeployment},
		{Name: "bastion_aws_machine_template.yaml.tmpl", Data: bastionAWSMachineTemplate},
	}
}
