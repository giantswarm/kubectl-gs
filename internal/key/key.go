package key

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/pkg/normalize"
)

const (
	// NameChars represents the character set used to generate resource names.
	// (does not contain 1 and l, to avoid confusion)
	NameChars = "023456789abcdefghijkmnopqrstuvwxyz"
	// NameLengthLong represents the number of characters used to create a resource name when --enable-long-names feature flag is used.
	NameLengthLong = 10
	// NameLengthShort represents the number of characters used to create a resource name.
	NameLengthShort = 5

	organizationNamespaceFormat = "org-%s"
)

const (
	RoleLabel                     = "role"
	SSHSSOPubKeyLabel             = "ssh-sso-public-key"
	GiantswarmNamespace           = "giantswarm"
	CertOperatorVersionKubeconfig = "0.0.0"
)

const (
	// FirstAWSOrgNamespaceRelease is the first GS release that creates Clusters in Org Namespaces by default
	FirstAWSOrgNamespaceRelease = "16.0.0"
)

func BastionSSHDConfigEncoded() string {
	return base64.StdEncoding.EncodeToString([]byte(bastionSSHDConfig))
}

func NodeSSHDConfigEncoded() string {
	return base64.StdEncoding.EncodeToString([]byte(nodeSSHDConfig))
}

func ValidateName(name string, enableLongNames bool) (bool, error) {
	maxLength := NameLengthShort
	if enableLongNames {
		maxLength = NameLengthLong
	}

	pattern := fmt.Sprintf("^[a-z][a-z0-9]{0,%d}$", maxLength-1)
	matched, err := regexp.MatchString(pattern, name)
	return matched, microerror.Mask(err)
}

func GenerateName(enableLongNames bool) (string, error) {
	for {
		letterRunes := []rune(NameChars)
		length := NameLengthShort
		if enableLongNames {
			length = NameLengthLong
		}
		characters := make([]rune, length)
		r := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404
		for i := range characters {
			characters[i] = letterRunes[r.Intn(len(letterRunes))] //nolint:gosec
		}

		generatedName := string(characters)

		if valid, err := ValidateName(generatedName, enableLongNames); err != nil {
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

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// IsOrgNamespaceVersion returns whether a given AWS GS Release Version is based on clusters in Org Namespace
func IsOrgNamespaceVersion(version string) bool {
	// TODO: this has to return true as soon as v16.0.0 is the newest version
	// Background: in case the release version is not set, aws-admission-controller mutates to the the latest AWS version,
	// see https://github.com/giantswarm/aws-admission-controller/blob/ef83d90fc856fbc0484bec967064834c0b8d2c1e/pkg/aws/v1alpha3/cluster/mutate_cluster.go#L191-L202
	// so as soon as the latest version is >=16.0.0 we are going to need the org-namespace as default here.
	if version == "" {
		return true
	}
	OrgNamespaceVersion, _ := semver.New(FirstAWSOrgNamespaceRelease)
	releaseVersion, _ := semver.New(version)
	return releaseVersion.GE(*OrgNamespaceVersion)
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
		return "", microerror.Maskf(unmashalToMapFailedError, err.Error())
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
		return nil, microerror.Maskf(unmashalToMapFailedError, err.Error())
	}

	return data, nil
}

func SSHSSOPublicKey(ctx context.Context, client runtimeclient.Client) (string, error) {
	secretList := &v1.SecretList{}
	err := client.List(
		ctx,
		secretList,
		runtimeclient.MatchingLabels{
			RoleLabel: SSHSSOPubKeyLabel,
		},
		runtimeclient.InNamespace(GiantswarmNamespace))
	if err != nil {
		return "", microerror.Mask(err)
	}
	if len(secretList.Items) == 0 {
		return "", microerror.Mask(fmt.Errorf("failed to find secret with ssh sso public key in MC"))
	}

	sshSSOPublicKey := string(secretList.Items[0].Data["value"])

	return sshSSOPublicKey, nil
}

func GetReleaseComponents(ctx context.Context, client runtimeclient.Client, releaseName string) (map[string]string, error) {
	releaseComponents := make(map[string]string)
	release := &releasev1alpha1.Release{}
	err := client.Get(ctx, runtimeclient.ObjectKey{Namespace: "", Name: fmt.Sprintf("v%s", releaseName)}, release)
	if err != nil {
		return releaseComponents, microerror.Mask(err)
	}

	for _, component := range release.Spec.Components {
		releaseComponents[component.Name] = component.Version
	}

	return releaseComponents, nil
}

func UbuntuSudoersConfigEncoded() string {
	return base64.StdEncoding.EncodeToString([]byte(ubuntuSudoersConfig))
}

func OrganizationNamespaceFromName(name string) string {
	name = normalize.AsDNSLabelName(fmt.Sprintf(organizationNamespaceFormat, name))

	return name
}

func AzureStorageAccountTypeForVMSize(vmSize string) string {
	// Ugly but hardcoding is the only way to know premium storage support without reaching Azure API.
	vmTypesWithPremiumStorageSupport := map[string]bool{
		"Standard_B12ms": true, "Standard_B16ms": true, "Standard_B20ms": true,
		"Standard_F2s": true, "Standard_F4s": true, "Standard_F8s": true, "Standard_F16s": true,
		"Standard_D4s_v3": true, "Standard_D8s_v3": true, "Standard_D16s_v3": true, "Standard_D32s_v3": true, "Standard_D48s_v3": true, "Standard_D64s_v3": true,
		"Standard_E4-2s_v3": true, "Standard_E4s_v3": true, "Standard_E8-2s_v3": true, "Standard_E8-4s_v3": true, "Standard_E8s_v3": true, "Standard_E16-4s_v3": true, "Standard_E16-8s_v3": true,
		"Standard_E16s_v3": true, "Standard_E20s_v3": true, "Standard_E32-8s_v3": true, "Standard_E32-16s_v3": true, "Standard_E32s_v3": true,
		"Standard_PB6s":     true,
		"Standard_E4-2s_v4": true, "Standard_E4s_v4": true, "Standard_E8-2s_v4": true, "Standard_E8-4s_v4": true, "Standard_E8s_v4": true, "Standard_E16-4s_v4": true, "Standard_E16-8s_v4": true, "Standard_E16s_v4": true, "Standard_E20s_v4": true, "Standard_E32-8s_v4": true, "Standard_E32-16s_v4": true, "Standard_E32s_v4": true, "Standard_E48s_v4": true, "Standard_E64-16s_v4": true, "Standard_E64-32s_v4": true, "Standard_E64s_v4": true,
		"Standard_E4-2ds_v4": true, "Standard_E4ds_v4": true, "Standard_E8-2ds_v4": true, "Standard_E8-4ds_v4": true, "Standard_E8ds_v4": true, "Standard_E16-4ds_v4": true, "Standard_E16-8ds_v4": true, "Standard_E16ds_v4": true, "Standard_E20ds_v4": true, "Standard_E32-8ds_v4": true, "Standard_E32-16ds_v4": true, "Standard_E32ds_v4": true, "Standard_E48ds_v4": true, "Standard_E64-16ds_v4": true, "Standard_E64-32ds_v4": true, "Standard_E64ds_v4": true,
		"Standard_D4ds_v4": true, "Standard_D8ds_v4": true, "Standard_D16ds_v4": true, "Standard_D32ds_v4": true, "Standard_D48ds_v4": true, "Standard_D64ds_v4": true,
		"Standard_D4s_v4": true, "Standard_D8s_v4": true, "Standard_D16s_v4": true, "Standard_D32s_v4": true, "Standard_D48s_v4": true, "Standard_D64s_v4": true,
		"Standard_F4s_v2": true, "Standard_F8s_v2": true, "Standard_F16s_v2": true, "Standard_F32s_v2": true, "Standard_F48s_v2": true, "Standard_F64s_v2": true, "Standard_F72s_v2": true,
		"Standard_NV12s_v3": true, "Standard_NV24s_v3": true, "Standard_NV48s_v3": true,
		"Standard_L8s_v2": true, "Standard_L16s_v2": true, "Standard_L32s_v2": true, "Standard_L48s_v2": true, "Standard_L64s_v2": true, "Standard_L80s_v2": true,
		"Standard_M208s_v2": true, "Standard_M416-208s_v2": true, "Standard_M416s_v2": true, "Standard_M416-208ms_v2": true, "Standard_M416ms_v2": true,
		"Standard_NV4as_v4": true, "Standard_NV8as_v4": true, "Standard_NV16as_v4": true, "Standard_NV32as_v4": true,
		"Standard_D4as_v4": true, "Standard_D8as_v4": true, "Standard_D16as_v4": true, "Standard_D32as_v4": true, "Standard_D48as_v4": true, "Standard_D64as_v4": true, "Standard_D96as_v4": true,
		"Standard_E4as_v4": true, "Standard_E8as_v4": true, "Standard_E16as_v4": true, "Standard_E20as_v4": true, "Standard_E32as_v4": true, "Standard_E48as_v4": true, "Standard_E64as_v4": true, "Standard_E96as_v4": true,
	}
	if vmTypesWithPremiumStorageSupport[vmSize] {
		return "Premium_LRS"
	}
	return "Standard_LRS"
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
