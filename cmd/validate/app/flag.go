package app

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagCatalog       = "catalog"
	flagCluster       = "cluster"
	flagName          = "name"
	flagNamespace     = "namespace"
	flagUserConfigMap = "user-configmap"
	flagUserSecret    = "user-secret"
	flagVersion       = "version"
)

type flag struct {
	Catalog           string
	Cluster           string
	Name              string
	Namespace         string
	flagUserConfigMap string
	flagUserSecret    string
	Version           string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog name where app is stored.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "App name.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace where the app will be deployed.")
	cmd.Flags().StringVar(&f.Cluster, flagCluster, "", "Cluster where the app will be deployed.")
	cmd.Flags().StringVar(&f.flagUserConfigMap, flagUserConfigMap, "", "Path to the user app configmap file data.")
	cmd.Flags().StringVar(&f.flagUserSecret, flagUserSecret, "", "Path to the user app secret file data.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to be installed.")
}

func (f *flag) Validate() error {

	if f.Catalog == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCatalog)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.Namespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNamespace)
	}
	if f.Cluster == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCluster)
	}
	if f.Version == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagVersion)
	}

	return nil
}
