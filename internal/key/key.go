package key

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/pkg/normalize"
)

const (
	// IDChars represents the character set used to generate cluster IDs.
	// (does not contain 1 and l, to avoid confusion)
	IDChars = "023456789abcdefghijkmnopqrstuvwxyz"
	// IDLength represents the number of characters used to create a cluster ID.
	IDLength = 5

	organizationNamespaceFormat = "org-%s"
)

func GenerateID() string {
	compiledRegexp, _ := regexp.Compile("^[a-z]+$")

	/* #nosec G404 */
	for {
		letterRunes := []rune(IDChars)
		b := make([]rune, IDLength)
		rand.Seed(time.Now().UnixNano())
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}

		id := string(b)

		if _, err := strconv.Atoi(id); err == nil {
			// ID is made up of numbers only, which we want to avoid.
			continue
		}

		matched := compiledRegexp.MatchString(id)
		if matched {
			// ID is made up of letters only, which we also avoid.
			continue
		}

		return id
	}
}

func GenerateAssetName(values ...string) string {
	return strings.Join(values, "-")
}

// IsCAPAVersion returns whether a given GS Release Version is based on the CAPI/CAPA projects
// TODO: make this a >= comparison
func IsCAPAVersion(version string) bool {
	return version == "20.0.0"
}

// readConfigMapFromFile reads a configmap from a YAML file.
func ReadConfigMapYamlFromFile(fs afero.Fs, path string) (string, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", microerror.Mask(err)
	}

	rawMap := map[string]interface{}{}
	err = yaml.Unmarshal(data, &rawMap)
	if err != nil {
		return "", microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	return string(data), nil
}

// readSecretFromFile reads a configmap from a YAML file.
func ReadSecretYamlFromFile(fs afero.Fs, path string) ([]byte, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = yaml.Unmarshal(data, &map[string]interface{}{})
	if err != nil {
		return nil, microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	return data, nil
}

func OrganizationNamespaceFromName(name string) string {
	name = normalize.AsDNSLabelName(fmt.Sprintf(organizationNamespaceFormat, name))

	return name
}
