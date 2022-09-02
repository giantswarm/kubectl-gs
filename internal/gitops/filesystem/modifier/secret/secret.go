package secret

import (
	"encoding/base64"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/helper"
)

type SecretModifier struct {
	KeysToAdd map[string]string

	secret map[string]interface{}
}

func (sec SecretModifier) Execute(rawYaml []byte) ([]byte, error) {
	err := helper.Unmarshal(rawYaml, &sec.secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	sec.addKeys()

	return helper.Marshal(sec.secret)
}

func (sec *SecretModifier) addKeys() {
	if sec.secret["data"] == nil {
		sec.secret["data"] = map[string]interface{}{}
	}

	for k, v := range sec.KeysToAdd {
		if _, ok := sec.secret["data"].(map[string]interface{})[k]; ok {
			continue
		}

		sec.secret["data"].(map[string]interface{})[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
}
