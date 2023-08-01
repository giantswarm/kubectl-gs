package aws

import (
	_ "embed"
)

//go:embed cluster_eks.yaml.tmpl
var clusterEKS string

//go:embed aws_managed_control_plane.yaml.tmpl
var awsManagedControlPlane string

//go:embed aws_cluster_role_identity.yaml.tmpl
var awsClusterRoleIdentityName string

type Template struct {
	Name string
	Data string
}

// GetEKSTemplate merges .tmpl files for an EKS cluster.
func GetEKSTemplates() []Template {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []Template{
		{Name: "cluster_eks.yaml.tmpl", Data: clusterEKS},
		{Name: "aws_managed_control_plane.yaml.tmpl", Data: awsManagedControlPlane},
		{Name: "aws_cluster_role_identity.yaml.tmpl", Data: awsClusterRoleIdentityName},
	}
}
