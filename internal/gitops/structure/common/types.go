package common

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/encryption"
)

type StructureConfig struct {
	App                    string
	AppBase                string
	AppInstallTimeout      *metav1.Duration
	AppName                string
	AppNamespace           string
	AppOCIURLPrefix        string
	AppRollbackTimeout     *metav1.Duration
	AppUninstallTimeout    *metav1.Duration
	AppUpgradeTimeout      *metav1.Duration
	AppUserValuesConfigMap string
	AppUserValuesSecret    string
	AppVersion             string
	AppVersionRange        string
	// AppResources holds the OCIRepository and HelmRelease of the app, when
	// it doesn't come from a base.
	AppResources AppResources

	Provider             string
	ClusterBaseTemplates ClusterBaseTemplates

	ClusterBase       string
	Release           string
	ClusterUserConfig string
	InCluster         string

	EncryptionKeyPair encryption.KeyPair
	EncryptionTarget  string

	ManagementCluster string
	Organization      string
	RepositoryName    string
	SkipEncryption    bool
	WorkloadCluster   string

	// KeepSingleKustomization adds the cluster definition to a workload
	// cluster that already has a single Kustomization applying its whole
	// directory, instead of giving the cluster a Kustomization of its own.
	// Moving the apps to a second Kustomization would make Flux uninstall them
	// before the new one installs them again.
	KeepSingleKustomization bool

	Region string

	// Azure only
	AzureSubscriptionID string

	// PreventDeletion labels the resources of a cluster base and sets
	// global.metadata.preventDeletion in its values.
	PreventDeletion bool
}

type AppResources struct {
	HelmRelease   string
	OCIRepository string
}

type ClusterBaseTemplates struct {
	// ClusterResources holds the OCIRepository and HelmRelease of the cluster.
	ClusterResources string
	ClusterValues    string
}

type Template struct {
	Data       string
	Name       string
	Permission os.FileMode
}
