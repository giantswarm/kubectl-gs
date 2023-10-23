package helper

import (
	sops "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/cmd/sops/formats"
	"github.com/getsops/sops/v3/decrypt"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

func Marshal(mapYaml map[string]interface{}) ([]byte, error) {
	rawYaml, err := yaml.Marshal(mapYaml)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return rawYaml, nil
}

func Unmarshal(rawYaml []byte, mapYaml *map[string]interface{}) error {
	decRawYaml, err := decrypt.DataWithFormat(rawYaml, formats.Yaml)
	if err == sops.MetadataNotFound {
		decRawYaml = rawYaml
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = yaml.Unmarshal(decRawYaml, mapYaml)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
