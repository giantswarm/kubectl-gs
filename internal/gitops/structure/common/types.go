package common

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/encryption"
)

type StructureConfig struct {
	App                    string
	AppBase                string
	AppCatalog             string
	AppInstallTimeout      *metav1.Duration
	AppName                string
	AppNamespace           string
	AppRollbackTimeout     *metav1.Duration
	AppUninstallTimeout    *metav1.Duration
	AppUpgradeTimeout      *metav1.Duration
	AppUserValuesConfigMap string
	AppUserValuesSecret    string
	AppVersion             string
	AppVersionRepository   string

	Provider             string
	ClusterBaseTemplates ClusterBaseTemplates

	ClusterBase       string
	Release           string
	ClusterUserConfig string
	InCluster         bool

	EncryptionKeyPair encryption.KeyPair
	EncryptionTarget  string

	ManagementCluster string
	Organization      string
	RepositoryName    string
	SkipMAPI          bool
	WorkloadCluster   string

	Region string

	// Azure only
	AzureSubscriptionID string
}

type ClusterBaseTemplates struct {
	ClusterAppCr  string
	ClusterValues string
}

type Template struct {
	Data       string
	Name       string
	Permission os.FileMode
}
