// Package sigskus modifies the SOPS configuration, .sops.yaml, of a GitOps
// repository.
package sigskus

import (
	"errors"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/helper"
)

var masterKeyNotFoundError = &microerror.Error{
	Kind: "masterKeyNotFoundError",
}

// IsMasterKeyNotFound asserts masterKeyNotFoundError.
func IsMasterKeyNotFound(err error) bool {
	return errors.Is(err, masterKeyNotFoundError)
}

// MasterKeyRuleModifier encrypts the files PathRegex matches with the key of
// the rule for MasterPathRegex, the management cluster's master key. Its rule
// goes ahead of every other rule, as sops uses the first rule whose path
// matches, so an organisation key's rule for the same directory can't take
// them.
type MasterKeyRuleModifier struct {
	MasterPathRegex string
	PathRegex       string
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
func (m MasterKeyRuleModifier) Execute(rawYaml []byte) ([]byte, error) {
	var config map[string]interface{}
	err := helper.Unmarshal(rawYaml, &config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	rules, _ := config["creation_rules"].([]interface{})
	if findRule(rules, m.PathRegex) != nil {
		return rawYaml, nil
	}

	master := findRule(rules, m.MasterPathRegex)
	if master == nil {
		return nil, microerror.Maskf(masterKeyNotFoundError, "no rule for %s in .sops.yaml, so there is no master key to encrypt %s with", m.MasterPathRegex, m.PathRegex)
	}

	rule := map[string]interface{}{}
	for k, v := range master {
		rule[k] = v
	}
	rule["path_regex"] = m.PathRegex

	config["creation_rules"] = append([]interface{}{rule}, rules...)

	return helper.Marshal(config)
}

// HasMasterKey reports whether the .sops.yaml has a rule for masterPathRegex.
func HasMasterKey(rawYaml []byte, masterPathRegex string) (bool, error) {
	var config map[string]interface{}
	err := helper.Unmarshal(rawYaml, &config)
	if err != nil {
		return false, microerror.Mask(err)
	}

	rules, _ := config["creation_rules"].([]interface{})

	return findRule(rules, masterPathRegex) != nil, nil
}

func findRule(rules []interface{}, pathRegex string) map[string]interface{} {
	for _, r := range rules {
		rule, ok := r.(map[string]interface{})
		if ok && rule["path_regex"] == pathRegex {
			return rule
		}
	}

	return nil
}
