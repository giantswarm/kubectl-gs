package sigskus

import (
	"reflect"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/modifier/helper"
)

const (
	encryptedRegex = "^(data|stringData)$"
)

type SopsModifier struct {
	RulesToAdd []map[string]interface{}

	config map[string]interface{}
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
func (sops SopsModifier) Execute(rawYaml []byte) ([]byte, error) {
	err := helper.Unmarshal(rawYaml, &sops.config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	sops.addRules()

	return helper.Marshal(sops.config)
}

func NewRule(encrypt, path, fingerprint string) map[string]interface{} {
	rule := map[string]interface{}{
		"encrypted_regex": encrypt,
		"path_regex":      path,
		"pgp":             fingerprint,
	}

	if encrypt == "" {
		rule["encrypted_regex"] = encryptedRegex
	}

	return rule
}

func (sops *SopsModifier) addRules() {
	for _, r := range sops.RulesToAdd {
		if !sops.isPresent(r) {
			sops.config["creation_rules"] = append(sops.config["creation_rules"].([]interface{}), r)
		}
	}
}

func (sops *SopsModifier) isPresent(rule map[string]interface{}) bool {
	for _, r := range sops.config["creation_rules"].([]interface{}) {
		if reflect.DeepEqual(r, rule) {
			return true
		}
	}

	return false
}
