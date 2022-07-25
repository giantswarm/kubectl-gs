package app

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagManagementCluster = "management-cluster"
	flagName              = "name"
	flagOrganization      = "organization"
	flagWorkloadCluster   = "workload-cluster"

	// App CR part
	flagApp                 = "app"
	flagBase                = "base"
	flagCatalog             = "catalog"
	flagNamespace           = "namespace"
	flagUserValuesConfigMap = "user-configmap"
	flagUserValuesSecret    = "user-secret"
	flagVersion             = "version"
)

type flag struct {
	ManagementCluster string
	Name              string
	Organization      string
	WorkloadCluster   string

	// App CR part
	App                 string
	Base                string
	Catalog             string
	Namespace           string
	UserValuesConfigMap string
	UserValuesSecret    string
	Version             string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the Management Cluster the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.WorkloadCluster, flagWorkloadCluster, "", "Name of the Workload Cluster to configure app to.")

	//CAPx only
	cmd.Flags().StringVar(&f.App, flagApp, "", "App from the catalog to install.")
	cmd.Flags().StringVar(&f.Base, flagBase, "", "Path to the base directory. It must be relative to the repository root.")
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog to install the app from.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace to install app into.")
	cmd.Flags().StringVar(&f.UserValuesConfigMap, flagUserValuesConfigMap, "", "Values to customize the app with as a ConfigMap.")
	cmd.Flags().StringVar(&f.UserValuesSecret, flagUserValuesSecret, "", "Values to customize the app with as a Secret.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to install.")
}

func (f *flag) Validate() error {
	if f.ManagementCluster == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagManagementCluster)
	}

	if f.Organization == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagOrganization)
	}

	if f.WorkloadCluster == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagWorkloadCluster)
	}

	if f.App == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagApp)
	}

	if f.Catalog == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagCatalog)
	}

	if f.Namespace == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagNamespace)
	}

	if f.Version == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagVersion)
	}

	return nil
}
