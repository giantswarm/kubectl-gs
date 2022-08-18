package common

import (
	"os"

	"github.com/giantswarm/kubectl-gs/internal/gitops/encryption"
)

type StructureConfig struct {
	App                    string
	AppBase                string
	AppCatalog             string
	AppName                string
	AppNamespace           string
	AppUserValuesConfigMap string
	AppUserValuesSecret    string
	AppVersion             string
	AppVersionRepository   string

	ClusterBase           string
	ClusterRelease        string
	ClusterUserConfig     string
	DefaultAppsRelease    string
	DefaultAppsUserConfig string

	EncryptionKeyPair encryption.KeyPair
	EncryptionTarget  string

	ManagementCluster string
	Organization      string
	RepositoryName    string
	WorkloadCluster   string
}

type Template struct {
	Data       string
	Name       string
	Permission os.FileMode
}
