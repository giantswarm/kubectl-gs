package key

import (
	"fmt"
)

const (
	appCRFile                 = "appcr.yaml"
	appsDirectory             = "apps"
	appUserConfigPatchFile    = "patch_app_userconfig.yaml"
	automaticUpdatesDirectory = "automatic-updates"
	clusterDirectory          = "cluster"
	configMapFile             = "configmap.yaml"
	imagePolicyFile           = "imagepolicy.yaml"
	imageRepositoryFile       = "imagerepository.yaml"
	kustomizationFile         = "kustomization.yaml"
	organizationsDirectory    = "organizations"
	secretFile                = "secret.yaml"
	secretsDirectory          = "secrets"
	sopsKeysDirectory         = ".sops.keys"
	workloadClusterDirectory  = "workload-clusters"

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

func ImagePolicyFileName() string {
	return imagePolicyFile
}

func ImageRepositoryFileName() string {
	return imagePolicyFile
}

func KustomizationFileName() string {
	return kustomizationFile
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

func SopsKeysDirName() string {
	return sopsKeysDirectory
}

func WorkloadClustersDirName() string {
	return workloadClusterDirectory
}

// GetMcDir points to `MC_NAME` directory, first of
// the three main repository layers.
func McDirPath(mc string) string {
	return fmt.Sprintf(mcDirectoryTemplate, mc)
}

// GetOrgDir points to `MC_NAME` directory, second of
// the three main repository layers.
func OrgDirPath(mc, org string) string {
	return fmt.Sprintf(orgDirectoryTemplate, mc, org)
}

// GetResource is general function returning path to
// a given resource. The rationale behind it is there are
// many resources that appear in multiple places like `appcr.yaml`,
// or `secret`, so having a general function is better than
// having N of them, each for a dedicated layers.
func ResourcePath(path, name string) string {
	return fmt.Sprintf("%s/%s", path, name)
}

// GetWcDir points to `MC_NAME` directory, third of
// the three main repository layers.
func WcDirPath(mc, org, wc string) string {
	return fmt.Sprintf(wcDirectoryTemplate, mc, org, wc)
}
