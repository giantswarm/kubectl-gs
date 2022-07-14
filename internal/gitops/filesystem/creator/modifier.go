package creator

import (
	"bytes"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	jmespath "github.com/jmespath/go-jmespath"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

const (
	kustomizeResourceRemove = "resources[?@ != '%s'"
	kustomizeResourceSearch = "resources[] | !contains(@, '%s')"
)

// My idea for now was to create some sort of modifiers
// that consume []byte input, modify it accordingly to the
// provided configuration, and return []byte that Creator can
// write to a file.
type Modifier interface {
	Execute([]byte) ([]byte, error)
}

type KustomizationModifier struct {
	ResourcesToAdd []string
}

func (km KustomizationModifier) Execute(rawYaml []byte) ([]byte, error) {
	var err error

	// This is needed to use JmesPath
	var structJson interface{}
	rawJson, err := yaml.YAMLToJSON(rawYaml)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// This is needed to use JmesPath
	err = getUnmarshalled(rawJson, &structJson)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// getResourcesListMods get list of JSONPatch operations
	// to perform on a given file
	resMods, err := km.getResourcesListMods(structJson)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// applyResourcesListMods applies the list of JSONPatch
	// operations from the previous step
	rawYaml, err = km.applyResourcesListMods(rawJson, resMods)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return rawYaml, nil
}

func (km KustomizationModifier) applyResourcesListMods(input, ops []byte) ([]byte, error) {
	patch, err := jsonpatch.DecodePatch(ops)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	modJson, err := patch.Apply(input)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	input, err = yaml.JSONToYAML(modJson)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return input, nil
}

// The idea for getResourcesListMods method is to go through certain
// KustomizationModifier lists for adding, deleting, replacing resources,
// and based on the JmesPath search outcome produce list of JsonPatch
// operations.
func (km KustomizationModifier) getResourcesListMods(structJson interface{}) ([]byte, error) {
	operations := make([][]byte, 0)

	for _, r := range km.ResourcesToAdd {
		ok, err := jmespath.Search(fmt.Sprintf(kustomizeResourceSearch, r), structJson)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if ok.(bool) {
			operations = append(operations, addKustomizationResources(r))
		}
	}

	operations = [][]byte{
		[]byte(`[`),
		bytes.Join(operations, []byte(`,`)),
		[]byte(`]`),
	}

	return bytes.Join(operations, []byte(``)), nil
}

func addKustomizationResources(resource string) []byte {
	return []byte(`{ "op": "add", "path": "/resources/-", "value": "` + resource + `"}`)
}

func getUnmarshalled(src []byte, dest *interface{}) error {
	err := json.Unmarshal(src, dest)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
