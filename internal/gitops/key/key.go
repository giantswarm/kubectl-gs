package key

import (
	"fmt"
)

const (
	appCRFile                  = "appcr.yaml"
	appsDirectory              = "apps"
	automaticUpdatesDirectory  = "automatic-updates"
	clusterDirectory           = "cluster"
	configMapFile              = "configmap.yaml"
	encryptionRegex            = "%s/%s.*\\.enc\\.yaml"
	fluxKustomizationFile      = "%s.yaml"
	imagePolicyFile            = "imagepolicy.yaml"
	imageRepositoryFile        = "imagerepository.yaml"
	managementClusterDirectory = "management-clusters"
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
	return fmt.Sprintf(encryptionRegex, base, target)
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
