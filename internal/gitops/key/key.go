package key

import (
	"fmt"
	"strings"
)

const (
	appCRFile                  = "appcr.yaml"
	appsDirectory              = "apps"
	automaticUpdatesDirectory  = "automatic-updates"
	clusterDirectory           = "cluster"
	configMapFile              = "configmap.yaml"
	encryptionRegex            = "%s/.*\\.enc\\.yaml"
	fluxKustomizationFile      = "%s.yaml"
	imagePolicyFile            = "imagepolicy.yaml"
	imageRepositoryFile        = "imagerepository.yaml"
	managementClusterDirectory = "management-clusters"
	mapiDirectory              = "mapi"
	organizationsDirectory     = "organizations"
	secretFile                 = "secret.yaml"
	secretsDirectory           = "secrets"
	sigsKustomizationFile      = "kustomization.yaml"
	sopsConfigFile             = ".sops.yaml"
	sopsKeysDirectory          = ".sops.keys"
	sopsKeyName                = "%s.%s.asc"
	sopsSecret                 = "sops-gpg-%s"        //#nosec G101 -- false positive (no secret here)
	sopsSecretFile             = "%s.gpgkey.enc.yaml" //#nosec G101 -- false positive (no secret here)
	workloadClusterDirectory   = "workload-clusters"

	baseTemplatesDirectory        = "bases"
	clusterBaseTemplatesDirectory = "clusters"
	clusterBaseTemplate           = baseTemplatesDirectory + "/" + clusterBaseTemplatesDirectory + "/%s"

	mcDirectoryTemplate  = "management-clusters/%s"
	orgDirectoryTemplate = "%s/organizations/%s"
	wcDirectoryTemplate  = "%s/workload-clusters/%s"
)

func AppCRFileName() string {
	return appCRFile
}

func AppsDirName() string {
	return appsDirectory
}

func AutoUpdatesDirName() string {
	return automaticUpdatesDirectory
}

func BaseTemplatesDirName() string {
	return baseTemplatesDirectory
}

func ClusterBaseTemplatesDirName() string {
	return clusterBaseTemplatesDirectory
}

func ClusterBaseTemplatesPath() string {
	return fmt.Sprintf("%s/%s", BaseTemplatesDirName(), ClusterBaseTemplatesDirName())
}

func ClusterBasePath(provider string) string {
	return fmt.Sprintf(clusterBaseTemplate, provider)
}

// BaseDirPath is one of the most important functions that
// constructs the base path to the layer currently being
// configure.
func BaseDirPath(mc, org, wc string) string {
	base := fmt.Sprintf(mcDirectoryTemplate, mc)

	if org != "" {
		base = fmt.Sprintf(orgDirectoryTemplate, base, org)
	}

	if wc != "" {
		base = fmt.Sprintf(wcDirectoryTemplate, base, wc)
	}

	return base
}

func ClusterDirName() string {
	return clusterDirectory
}

func ConfigMapFileName() string {
	return configMapFile
}

func EncryptionRegex(base, target string) string {
	// Sanitize target to make it easier to work with
	target = strings.TrimPrefix(target, "/")
	target = strings.TrimSuffix(target, "/")

	// These conditions are here to prohibit misconfiguration of the
	// .sops.yaml, Kustomization CR and *.gpgkey.enc.yaml Secrets.
	// Context: user should be given some freedom towards directory
	// to encrypt / decrypt, meaning it shouldn't always be `secrets/`
	// dir. This however introduced a risk to "escaping" the desired layer
	// and misconfiguring Flux. For example, user can tell `kgs` that
	// he wants to encrypt Management Cluster layer only, but then pass
	// the path to the Workload Cluster's App secres to the `target` flag,
	// making `kgs` to drop configuration in the wrong places.
	// To block this we check for `organizations` and `workload-clusters`
	// directories being specified, which we know are essential to the structure,
	// and we restore the default value in that case.
	if strings.HasPrefix(target, organizationsDirectory) {
		target = secretsDirectory
	}
	if strings.HasPrefix(target, workloadClusterDirectory) {
		target = secretsDirectory
	}

	if target != "" {
		base = fmt.Sprintf("%s/%s", base, target)
	}

	return fmt.Sprintf(encryptionRegex, base)
}

func FluxKustomizationFileName(name string) string {
	return fmt.Sprintf(fluxKustomizationFile, name)
}

func ImagePolicyFileName() string {
	return imagePolicyFile
}

func ImageRepositoryFileName() string {
	return imageRepositoryFile
}

func ManagementClustersDirName() string {
	return managementClusterDirectory
}

func MapiDirName() string {
	return mapiDirectory
}

func OrganizationsDirName() string {
	return organizationsDirectory
}

// ResourcePath is general function returning path to
// a given resource. The rationale behind it is there are
// many resources that appear in multiple places like `appcr.yaml`,
// or `secret`, so having a general function is better than
// having N of them, each for a dedicated layers, with complicated
// names to indicate the layer they serve.
func ResourcePath(path, name string) string {
	return fmt.Sprintf("%s/%s", path, name)
}

func SecretFileName() string {
	return secretFile
}

func SecretsDirName() string {
	return secretsDirectory
}

func SigsKustomizationFileName() string {
	return sigsKustomizationFile
}

func SopsConfigFileName() string {
	return sopsConfigFile
}

func SopsKeysDirName() string {
	return sopsKeysDirectory
}

func SopsKeyName(name, fingerprint string) string {
	return fmt.Sprintf(sopsKeyName, name, fingerprint)
}

func SopsKeyPrefix(mc, wc string) string {
	prefix := mc

	if wc != "" {
		prefix = wc
	}

	return prefix
}

func SopsSecretFileName(name string) string {
	return fmt.Sprintf(sopsSecretFile, name)
}

func SopsSecretName(name string) string {
	return fmt.Sprintf(sopsSecret, name)
}

func WorkloadClustersDirName() string {
	return workloadClusterDirectory
}
