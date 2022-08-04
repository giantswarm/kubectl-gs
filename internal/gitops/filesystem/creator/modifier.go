package creator

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	jsonpatch "github.com/evanphx/json-patch"
	jmespath "github.com/jmespath/go-jmespath"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

const (
	appVersionLocator = `version:\s([v0-9.]*).*\n`
	appVersionUpdater = "version: $1 # {\"$$imagepolicy\": \"default:%s:tag\"}\n"

	arrayContains    = "%s[] | contains(@, '%s')"
	mapContainsValue = "%s[].%s | contains(@, '%s')"
)

// My idea for now was to create some sort of modifiers
// that consume []byte input, modify it accordingly to the
// provided configuration, and return []byte that Creator can
// write to a file.
type Modifier interface {
	Execute([]byte) ([]byte, error)
}

type AppModifier struct {
	ImagePolicy string
}

type KustomizationModifier struct {
	ResourcesToAdd    []string
	ResourcesToCheck  []string
	ResourcesToRemove []string
}

// PostExecute modifies App CRs after creating the necessary
// resources first.
func (am AppModifier) Execute(rawYaml []byte) ([]byte, error) {

	if am.ImagePolicy != "" {
		re := regexp.MustCompile(appVersionLocator)
		rawYaml = re.ReplaceAll(rawYaml, []byte(fmt.Sprintf(appVersionUpdater, am.ImagePolicy)))
	}

	return rawYaml, nil
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
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
		if ok, err := isPresent(r, "resources", structJson); ok && err == nil {
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

// addResource uses JSONPatch to add an element to an resources
// array of kustomization.yaml.
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

// addToArray returns JSONpatch for adding an element to an array.
func addToArray(path, resource string) []byte {
	return []byte(`[{ "op": "add", "path": "` + path + `/-", "value": "` + resource + `"}]`)
}

// getUnmarshalledJson returns unmarshalled JSON.
func getUnmarshalledJson(src []byte, dest *interface{}) error {
	err := json.Unmarshal(src, dest)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// isPresent uses JmesPath to check for element existance in
// within the array.
func isPresent(resource, path string, file interface{}) (bool, error) {
	ok, err := jmespath.Search(fmt.Sprintf(arrayContains, path, resource), file)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return ok.(bool), nil
}
