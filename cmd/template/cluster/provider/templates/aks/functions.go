package aks

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
		return "", fmt.Errorf("subscription ID is required for AKS (--azure-subscription-id)")
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

	if metadata, ok := flagConfigData["global"].(map[string]interface{})["metadata"].(map[string]interface{}); ok {
		metadata["preventDeletion"] = flagInputs.Global.Metadata.PreventDeletion
	}

	finalConfigString, err := yaml.Marshal(flagInputs)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return string(finalConfigString), nil
}
