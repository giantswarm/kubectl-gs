package aws

import (
	_ "embed"
)

//go:embed aws_machine_pool.yaml.tmpl
var awsMachinePool string

//go:embed machine_pool.yaml.tmpl
var machinePool string

//go:embed kubeadm_config.yaml.tmpl
var kubeadmConfig string

//go:embed machine_pool_eks.yaml.tmpl
var machinePoolEKS string

//go:embed aws_managed_machine_pool.yaml.tmpl
var awsManagedMachinePool string

// GetAWSTemplate merges .tmpl files for an AWS machine pool.
func GetAWSTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		machinePool,
		awsMachinePool,
		kubeadmConfig,
	}
}

// GetEKSTemplate merges .tmpl files for an EKS machine pool.
func GetEKSTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		awsManagedMachinePool,
		machinePoolEKS,
	}
}
