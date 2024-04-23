package capz

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

func GenerateClusterValues(flagInputs ClusterConfig) (string, error) {
	var flagConfigData map[string]interface{}

	if flagInputs.Global.ProviderSpecific.Location == "" {
		return "", fmt.Errorf("region is required (--region)")
	}
	if flagInputs.Global.ProviderSpecific.SubscriptionID == "" {
		return "", fmt.Errorf("subscription ID is required for Azure (--azure-subscription-id)")
	}

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

	finalConfigString, err := yaml.Marshal(flagInputs)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return string(finalConfigString), nil
}

func GenerateDefaultAppsValues(flagConfig DefaultAppsConfig) (string, error) {
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
