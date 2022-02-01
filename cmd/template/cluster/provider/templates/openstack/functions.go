package openstack

import (
	"github.com/giantswarm/microerror"
	"github.com/imdario/mergo"
	"sigs.k8s.io/yaml"
)

func GenerateClusterConfigMapValues(flagInputs ClusterConfig, fileInputs ClusterConfig) (string, error) {
	var flagConfigData map[string]interface{}

	{
		flagConfigYAML, err := yaml.Marshal(flagInputs)
		if err != nil {
			return "", microerror.Mask(err)
		}

		err = yaml.Unmarshal(flagConfigYAML, &flagConfigData)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	var fileConfigData map[string]interface{}

	{
		fileConfigYAML, err := yaml.Marshal(fileInputs)
		if err != nil {
			return "", microerror.Mask(err)
		}

		err = yaml.Unmarshal(fileConfigYAML, &flagConfigData)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	finalConfigData := fileConfigData
	err := mergo.Merge(&finalConfigData, &flagConfigData, mergo.WithOverride)
	if err != nil {
		return "", microerror.Mask(err)
	}

	finalConfigString, err := yaml.Marshal(finalConfigData)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return string(finalConfigString), nil
}

func GenerateDefaultAppsConfigMapValues(flagConfig DefaultAppsConfig) (string, error) {
	var flagConfigData map[string]interface{}

	{
		flagConfigYAML, err := yaml.Marshal(flagConfig)
		if err != nil {
			return "", microerror.Mask(err)
		}

		err = yaml.Unmarshal(flagConfigYAML, &flagConfigData)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	finalConfigString, err := yaml.Marshal(flagConfigData)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return string(finalConfigString), nil
}
