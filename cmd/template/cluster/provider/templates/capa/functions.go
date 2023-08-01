package capa

import (
	"fmt"
	"strings"

	gsannotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

func GenerateClusterValues(flagInputs ClusterConfig) (string, error) {
	if flagInputs.Connectivity.Topology.Mode != "" && flagInputs.Connectivity.Topology.Mode != gsannotation.NetworkTopologyModeGiantSwarmManaged && flagInputs.Connectivity.Topology.Mode != gsannotation.NetworkTopologyModeUserManaged && flagInputs.Connectivity.Topology.Mode != gsannotation.NetworkTopologyModeNone {
		return "", fmt.Errorf("invalid topology mode value %q", flagInputs.Connectivity.Topology.Mode)
	}
	if flagInputs.Connectivity.Topology.PrefixListID != "" && !strings.HasPrefix(flagInputs.Connectivity.Topology.PrefixListID, "pl-") {
		return "", fmt.Errorf("invalid AWS prefix list ID %q", flagInputs.Connectivity.Topology.PrefixListID)
	}
	if flagInputs.Connectivity.Topology.TransitGatewayID != "" && !strings.HasPrefix(flagInputs.Connectivity.Topology.TransitGatewayID, "tgw-") {
		return "", fmt.Errorf("invalid AWS transit gateway ID %q", flagInputs.Connectivity.Topology.TransitGatewayID)
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
