package key

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	runtimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
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

const (
	AWSBastionInstanceType = "t3.small"

	CAPIRoleLabel = "cluster.x-k8s.io/role"
	CAPARoleTag   = "tag:sigs.k8s.io/cluster-api-provider-aws/role"

	FlatcarAMIOwner      = "075585003325"
	FlatcarChinaAMIOwner = "306934455918"

	RoleBastion = "bastion"

	RoleLabel           = "role"
	SSHSSOPubKeyLabel   = "ssh-sso-public-key"
	GiantswarmNamespace = "giantswarm"

	ControllerRuntimeBurstValue = 200
)

const (
	// FirstOrgNamespaceRelease is the first GS release that creates Clusters in Org Namespaces by default
	FirstAWSOrgNamespaceRelease = "16.0.0"
)

func BastionResourceName(clusterName string) string {
	return fmt.Sprintf("%s-bastion", clusterName)
}

func BastionSSHDConfigEncoded() string {
	return base64.StdEncoding.EncodeToString([]byte(bastionSSHDConfig))
}

func NodeSSHDConfigEncoded() string {
	return base64.StdEncoding.EncodeToString([]byte(nodeSSHDConfig))
}

func CAPAClusterOwnedTag(clusterName string) string {
	return fmt.Sprintf("tag:sigs.k8s.io/cluster-api-provider-aws/cluster/%s", clusterName)
}

func FlatcarAWSAccountID(awsRegion string) string {
	if strings.Contains(awsRegion, "cn-") {
		return FlatcarChinaAMIOwner
	} else {
		return FlatcarAMIOwner
	}
}

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

func GetCAPAEnvVars() []string {
	return []string{"AWS_SUBNET", "AWS_CONTROL_PLANE_MACHINE_TYPE", "AWS_REGION", "AWS_SSH_KEY_NAME"}
}

func GetControlPlaneInstanceProfile(clusterID string) string {
	return fmt.Sprintf("control-plane-%s", clusterID)
}

func GetNodeInstanceProfile(machinePoolID string, clusterID string) string {
	return fmt.Sprintf("nodes-%s-%s", machinePoolID, clusterID)
}

// IsCAPAVersion returns whether a given GS Release Version is based on the CAPI/CAPA projects
// TODO: make this a >= comparison
func IsCAPAVersion(version string) bool {
	return version == "20.0.0"
}

// IsCAPZVersion returns whether a given GS Release Version is based on the CAPI/CAPZ projects
func IsCAPZVersion(version string) bool {
	return version == "20.0.0"
}

// IsOrgNamespaceVersion returns whether a given AWS GS Release Version is based on clusters in Org Namespace
func IsOrgNamespaceVersion(version string) bool {
	// TODO: this has to return true as soon as v16.0.0 is the newest version
	// Background: in case the release version is not set, aws-admission-controller mutates to the the latest AWS version,
	// see https://github.com/giantswarm/aws-admission-controller/blob/ef83d90fc856fbc0484bec967064834c0b8d2c1e/pkg/aws/v1alpha3/cluster/mutate_cluster.go#L191-L202
	// so as soon as the latest version is >=16.0.0 we are going to need the org-namespace as default here.
	if version == "" {
		return false
	}
	OrgNamespaceVersion, _ := semver.New(FirstAWSOrgNamespaceRelease)
	releaseVersion, _ := semver.New(version)
	return releaseVersion.GE(*OrgNamespaceVersion)
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

func SSHSSOPublicKey() (string, error) {
	c, err := runtimeconfig.GetConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}
	c.Burst = ControllerRuntimeBurstValue // to avoid throttling the request

	k8sClient, err := runtimeclient.New(c, runtimeclient.Options{})
	if err != nil {
		return "", microerror.Mask(err)
	}
	secretList := &v1.SecretList{}

	err = k8sClient.List(
		context.TODO(),
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

	sshSSOPublicKey := base64.StdEncoding.EncodeToString(secretList.Items[0].Data["value"])

	return sshSSOPublicKey, nil
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
