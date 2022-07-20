package creator

import (
	//"bytes"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	jmespath "github.com/jmespath/go-jmespath"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

const (
	arrayContains = "%s[] | contains(@, '%s')"
)

// My idea for now was to create some sort of modifiers
// that consume []byte input, modify it accordingly to the
// provided configuration, and return []byte that Creator can
// write to a file.
type Modifier interface {
	Execute([]byte) ([]byte, error)
}

type KustomizationModifier struct {
	ResourcesToAdd    []string
	ResourcesToRemove []string
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
	err = getUnmarshalledJson(rawJson, &structJson)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, r := range km.ResourcesToAdd {
		if ok, err := isPresent(r, "resources", structJson); ok == true && err == nil {
			continue
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		err = addResource(r, &rawJson)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	rawYaml, err = yaml.JSONToYAML(rawJson)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return rawYaml, nil
}

func addResource(resource string, file *[]byte) error {
	patch, err := jsonpatch.DecodePatch(addToArray("/resources", resource))
	if err != nil {
		return microerror.Mask(err)
	}

	*file, err = patch.Apply(*file)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func isPresent(resource, path string, file interface{}) (bool, error) {
	ok, err := jmespath.Search(fmt.Sprintf(arrayContains, path, resource), file)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return ok.(bool), nil
}

func addToArray(path, resource string) []byte {
	return []byte(`[{ "op": "add", "path": "` + path + `/-", "value": "` + resource + `"}]`)
}

func getUnmarshalledJson(src []byte, dest *interface{}) error {
	err := json.Unmarshal(src, dest)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
