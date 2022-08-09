package fluxkus

import (
	"reflect"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/helper"
)

type KustomizationModifier struct {
	DecryptionToAdd string

	kustomization map[string]interface{}
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
func (km KustomizationModifier) Execute(rawYaml []byte) ([]byte, error) {
	err := helper.Unmarshal(rawYaml, &km.kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	km.addDecryption()

	return helper.Marshal(km.kustomization)
}

func (km *KustomizationModifier) addDecryption() {
	if km.DecryptionToAdd == "" {
		return
	}

	decryption := map[string]interface{}{
		"provider": "sops",
		"secretRef": map[string]interface{}{
			"name": km.DecryptionToAdd,
		},
	}

	current, ok := km.kustomization["spec"].(map[string]interface{})["decryption"]

	if !ok {
		km.kustomization["spec"].(map[string]interface{})["decryption"] = decryption
		return
	}

	if reflect.DeepEqual(decryption, current) {
		return
	}

	km.kustomization["spec"].(map[string]interface{})["decryption"] = decryption
}
