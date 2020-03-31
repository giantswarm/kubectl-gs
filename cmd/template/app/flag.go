package app

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagCatalog           = "catalog"
	flagConfigMap         = "configmap"
	flagKubeConfigContext = "kubeconfig-context"
	flagKubeConfigSecret  = "kubeconfig-secret"
	flagName              = "name"
	flagNamespace         = "namespace"
	flagSecret            = "secret"
	flagUserConfigMap     = "user-configmap"
	flagUserSecret        = "user-secret"
	flagVersion           = "version"
)

type flag struct {
	Catalog           string
	ConfigMap         string
	KubeConfigContext string
	KubeConfigSecret  string
	Name              string
	Namespace         string
	Secret            string
	flagUserConfigMap string
	flagUserSecret    string
	Version           string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog name where app is stored.")
	cmd.Flags().StringVar(&f.ConfigMap, flagConfigMap, "", "Path to a default app configuration configmap file data.")
	cmd.Flags().StringVar(&f.KubeConfigContext, flagKubeConfigContext, "", "App KubeConfig context in case it is set in cluster mode.")
	cmd.Flags().StringVar(&f.KubeConfigSecret, flagKubeConfigSecret, "", "App KubeConfig secret in case it is set out cluster mode.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "App name.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "App namespace.")
	cmd.Flags().StringVar(&f.Secret, flagSecret, "", "Path to a default app configuration secret file data.")
	cmd.Flags().StringVar(&f.flagUserConfigMap, flagUserConfigMap, "", "Path to the user app configmap file data.")
	cmd.Flags().StringVar(&f.flagUserSecret, flagUserSecret, "", "Path to the user app secret file data.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to be installed.")
}

func (f *flag) Validate() error {

	if f.Catalog == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCatalog)
	}
	if f.KubeConfigContext == "" && f.KubeConfigSecret == "" {
		return microerror.Maskf(invalidFlagError, "--%s or --%s must not be empty", flagKubeConfigContext, flagKubeConfigSecret)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.Namespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNamespace)
	}
	if f.Version == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagVersion)
	}

	return nil
}
