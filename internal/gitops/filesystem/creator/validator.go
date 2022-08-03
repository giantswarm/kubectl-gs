package creator

import (
	"fmt"

	jmespath "github.com/jmespath/go-jmespath"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

const (
	arrayElementContains = "%s[] | join(',', @) | contains(@, '%s')"
	baseResource         = "../base"

	// Validation errors templates
	objectFromBaseTemplate = "Operation cannot be fulfilled on the object coming from a base. Operation should be conducted on the base instead."
)

// Same idea as for the Modifier
type Validator interface {
	Execute([]byte) error
}

type KustomizationValidator struct {
	ReferencesBase bool
}

// Execute is the interface used by the creator to execute post modifier.
// It accepts and returns raw bytes.
func (ks KustomizationValidator) Execute(rawYaml []byte) error {
	var err error

	// This is needed to use JmesPath
	var structJson interface{}
	rawJson, err := yaml.YAMLToJSON(rawYaml)
	if err != nil {
		return microerror.Mask(err)
	}

	// This is needed to use JmesPath
	err = getUnmarshalledJson(rawJson, &structJson)
	if err != nil {
		return microerror.Mask(err)
	}

	if ks.ReferencesBase {
		ok, err := hasMatchingElement("resources", baseResource, structJson)
		if err != nil {
			return microerror.Mask(err)
		}

		if ok {
			return microerror.Maskf(validationError, objectFromBaseTemplate)
		}
	}

	return nil
}

func hasMatchingElement(path, expression string, file interface{}) (bool, error) {
	ok, err := jmespath.Search(fmt.Sprintf(arrayElementContains, path, expression), file)
	if err != nil {
		fmt.Println("heh")
		return false, microerror.Mask(err)
	}

	return ok.(bool), nil
}
