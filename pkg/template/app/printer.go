package app

import (
	"github.com/ghodss/yaml"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
)

func printAppCR(appCR *v1alpha1.App, defaultingEnabled bool) ([]byte, error) {
	appCRYaml, err := yaml.Marshal(appCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rawAppCR := map[string]interface{}{}
	err = yaml.Unmarshal(appCRYaml, &rawAppCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	delete(rawAppCR, "status")

	metadata, ok := rawAppCR["metadata"].(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(executionFailedError, "failed to get metadata for app CR")
	}
	delete(metadata, "creationTimestamp")

	spec, ok := rawAppCR["spec"].(map[string]interface{})
	if !ok {
		return nil, microerror.Maskf(executionFailedError, "failed to get spec for app CR")
	}

	delete(spec, "install")
	delete(spec, "namespaceConfig")

	if defaultingEnabled {
		delete(spec, "config")
		spec["kubeConfig"] = map[string]bool{
			"inCluster": false,
		}
	}

	if appCR.Spec.UserConfig.ConfigMap.Name == "" && appCR.Spec.UserConfig.Secret.Name == "" {
		delete(spec, "userConfig")
	}

	outputYaml, err := yaml.Marshal(rawAppCR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return outputYaml, nil
}
