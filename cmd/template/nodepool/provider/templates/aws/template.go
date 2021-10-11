package aws

import (
	_ "embed"
)

//go:embed machine_pool_eks.yaml.tmpl
var machinePoolEKS string

//go:embed aws_managed_machine_pool.yaml.tmpl
var awsManagedMachinePool string

// GetEKSTemplate merges .tmpl files for an EKS cluster.
func GetEKSTemplates() []string {
	// Order is important here.
	// The order in this slice determines in which order files will be applied.
	return []string{
		awsManagedMachinePool,
		machinePoolEKS,
	}
}
