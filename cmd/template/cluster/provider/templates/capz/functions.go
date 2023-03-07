package capa

import (
	"fmt"
	"strings"

	gsannotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

func GenerateClusterValues(flagInputs ClusterConfig) (string, error) {
	if flagInputs.Network.TopologyMode != "" && flagInputs.Network.TopologyMode != gsannotation.NetworkTopologyModeGiantSwarmManaged && flagInputs.Network.TopologyMode != gsannotation.NetworkTopologyModeUserManaged && flagInputs.Network.TopologyMode != gsannotation.NetworkTopologyModeNone {
		return "", fmt.Errorf("invalid topology mode value %q", flagInputs.Network.TopologyMode)
	}
	if flagInputs.Network.PrefixListID != "" && !strings.HasPrefix(flagInputs.Network.PrefixListID, "pl-") {
		return "", fmt.Errorf("invalid AWS prefix list ID %q", flagInputs.Network.PrefixListID)
	}
	if flagInputs.Network.TransitGatewayID != "" && !strings.HasPrefix(flagInputs.Network.TransitGatewayID, "tgw-") {
		return "", fmt.Errorf("invalid AWS transit gateway ID %q", flagInputs.Network.TransitGatewayID)
	}

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
