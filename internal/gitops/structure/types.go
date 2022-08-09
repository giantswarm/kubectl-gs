package structure

import (
	"github.com/giantswarm/kubectl-gs/internal/gitops/encryption"
)

// TBD: as you can see some of the fields are the same accorss
// different structs, so instead of keeping N types with duplicates
// it could be considered to add one big struct, and simply neglecting
// fields that are not neccessary for the structure being added.

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
