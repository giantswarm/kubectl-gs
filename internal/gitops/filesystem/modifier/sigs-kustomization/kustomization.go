package sigskus

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/modifier/helper"
)

type empty struct{}

type KustomizationModifier struct {
	ResourcesToAdd []string

	kustomization map[string]interface{}
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
func (km KustomizationModifier) Execute(rawYaml []byte) ([]byte, error) {
	err := helper.Unmarshal(rawYaml, &km.kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	km.addResource()

	return helper.Marshal(km.kustomization)
}

// addResource goes through resources and adds them to the
// kustomization.yaml
func (km *KustomizationModifier) addResource() {
	resMap := getResourceMap(km.kustomization["resources"].([]interface{}))

	for _, r := range km.ResourcesToAdd {
		if _, ok := resMap[r]; !ok {
			km.kustomization["resources"] = append(km.kustomization["resources"].([]interface{}), r)
		}
	}
}

func getResourceMap(resArr []interface{}) map[string]empty {
	resMap := make(map[string]empty)

	for _, r := range resArr {
		resMap[r.(string)] = empty{}
	}

	return resMap
}
