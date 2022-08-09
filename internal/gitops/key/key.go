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
	sopsSecret                 = "sops-gpg-%s"        //#nosec G101 -- false positive
	sopsSecretFile             = "%s.gpgkey.enc.yaml" //#nosec G101 -- false positive
	workloadClusterDirectory   = "workload-clusters"

	mcDirectoryTemplate  = "management-clusters/%s"
	orgDirectoryTemplate = "management-clusters/%s/organizations/%s"
	wcDirectoryTemplate  = "management-clusters/%s/organizations/%s/workload-clusters/%s"
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

func ClusterDirName() string {
	return clusterDirectory
}

func ConfigMapFileName() string {
	return configMapFile
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

func SopsSecretFileName(name string) string {
	return fmt.Sprintf(sopsSecretFile, name)
}

func SopsSecretName(name string) string {
	return fmt.Sprintf(sopsSecret, name)
}

func WorkloadClustersDirName() string {
	return workloadClusterDirectory
}

// McDirPath points to `MC_NAME` directory, first of
// the three main repository layers.
func McDirPath(mc string) string {
	return fmt.Sprintf(mcDirectoryTemplate, mc)
}

// OrgDirPath points to `MC_NAME` directory, second of
// the three main repository layers.
func OrgDirPath(mc, org string) string {
	return fmt.Sprintf(orgDirectoryTemplate, mc, org)
}

// ResourcePath is general function returning path to
// a given resource. The rationale behind it is there are
// many resources that appear in multiple places like `appcr.yaml`,
// or `secret`, so having a general function is better than
// having N of them, each for a dedicated layers.
func ResourcePath(path, name string) string {
	return fmt.Sprintf("%s/%s", path, name)
}

// WcDirPath points to `MC_NAME` directory, third of
// the three main repository layers.
func WcDirPath(mc, org, wc string) string {
	return fmt.Sprintf(wcDirectoryTemplate, mc, org, wc)
}
