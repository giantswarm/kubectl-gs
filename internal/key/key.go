package key

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/pkg/normalize"
)

const (
	// NameChars represents the character set used to generate resource names.
	// (does not contain 1 and l, to avoid confusion)
	NameChars = "023456789abcdefghijkmnopqrstuvwxyz"
	// NameLengthMax represents the maximum number of characters that can be used to create a resource.
	NameLengthMax = 20
	// NameLengthDefault represents the number of characters used to randomly generated names
	NameLengthDefault = 10

	organizationNamespaceFormat = "org-%s"
)

const CertOperatorVersionKubeconfig = "0.0.0"

func ValidateName(name string) (bool, error) {
	maxLength := NameLengthMax

	pattern := fmt.Sprintf("^[a-z]([-a-z0-9]{0,%d}[a-z0-9])?$", maxLength-2)
	matched, err := regexp.MatchString(pattern, name)
	return matched, microerror.Mask(err)
}

func GenerateName() (string, error) {
	for {
		letterRunes := []rune(NameChars)
		length := NameLengthDefault

		characters := make([]rune, length)
		r := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404
		for i := range characters {
			characters[i] = letterRunes[r.Intn(len(letterRunes))] //nolint:gosec
		}

		generatedName := string(characters)

		if valid, err := ValidateName(generatedName); err != nil {
			return "", microerror.Mask(err)
		} else if !valid {
			continue
		}

		return generatedName, nil
	}
}

func GenerateAssetName(values ...string) string {
	return strings.Join(values, "-")
}

// IsPureCAPIProvider returns whether a given provider is purely based on or fully migrated to CAPI
func IsPureCAPIProvider(provider string) bool {
	return contains(PureCAPIProviders(), provider)
}

// IsCAPIProviderUsingReleases returns whether a given provider is a CAPI provider that uses new releases.
func IsCAPIProviderUsingReleases(provider string) bool {
	return contains(CAPIProvidersUsingReleases(), provider)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// ReadConfigMapYamlFromFile reads a configmap from a YAML file.
func ReadConfigMapYamlFromFile(fs afero.Fs, path string) (string, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", microerror.Mask(err)
	}

	rawMap := map[string]interface{}{}
	err = yaml.Unmarshal(data, &rawMap)
	if err != nil {
		return "", microerror.Maskf(unmashalToMapFailedError, "%s", err.Error())
	}

	return sanitize(string(data)), nil
}

// Sanitize input yaml files
//
// Trim spaces from end of lines, because sig.k8s.io/yaml@v1.3.0 outputs the whole file content
// as a raw, single line string if there are empty spaces at end of lines.
func sanitize(content string) string {
	var sanitizedContent []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		sanitizedLine := strings.TrimRight(line, " ")
		sanitizedContent = append(sanitizedContent, sanitizedLine)
	}

	return strings.Join(sanitizedContent, "\n")
}

// ReadSecretYamlFromFile reads a configmap from a YAML file.
func ReadSecretYamlFromFile(fs afero.Fs, path string) ([]byte, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = yaml.Unmarshal(data, &map[string]interface{}{})
	if err != nil {
		return nil, microerror.Maskf(unmashalToMapFailedError, "%s", err.Error())
	}

	return data, nil
}

func OrganizationNamespaceFromName(name string) string {
	name = normalize.AsDNSLabelName(fmt.Sprintf(organizationNamespaceFormat, name))

	return name
}

func GetCacheDir() (string, error) {
	rootDir, err := os.UserCacheDir()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return filepath.Join(rootDir, "kubectl-gs"), nil
}

func IsTTY() bool {
	fileInfo, _ := os.Stdout.Stat()

	return fileInfo.Mode()&os.ModeCharDevice != 0
}
