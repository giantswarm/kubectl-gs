package openstack

import (
	_ "embed"
)

//go:embed machine_deployment.yaml.tmpl
var machineDeployment string

//go:embed open_stack_machine_template.yaml.tmpl
var openstackMachineTemplate string

//go:embed kubeadm_config_template.yaml.tmpl
var kubeadmConfigTemplate string

// GetTemplate merges all .tmpl files.
func GetTemplates() []string {
	return []string{
		machineDeployment,
		openstackMachineTemplate,
		kubeadmConfigTemplate,
	}
}
