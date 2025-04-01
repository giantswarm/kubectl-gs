package app

import (
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagManagementCluster = "management-cluster"
	flagName              = "name"
	flagOrganization      = "organization"
	flagSkipMAPI          = "skip-mapi"
	flagWorkloadCluster   = "workload-cluster"

	// App CR part
	flagApp                 = "app"
	flagBase                = "base"
	flagCatalog             = "catalog"
	flagInCluster           = "in-cluster"
	flagInstallTimeout      = "install-timeout"
	flagNamespace           = "namespace"
	flagRollbackTimeout     = "rollback-timeout"
	flagTargetNamespace     = "target-namespace"
	flagUninstallTimeout    = "uninstall-timeout"
	flagUpgradeTimeout      = "upgrade-timeout"
	flagUserValuesConfigMap = "user-configmap"
	flagUserValuesSecret    = "user-secret"
	flagVersion             = "version"
)

type flag struct {
	ManagementCluster string
	Name              string
	Organization      string
	SkipMAPI          bool
	WorkloadCluster   string

	// App CR part
	App                 string
	Base                string
	Catalog             string
	InCluster           bool
	InstallTimeout      time.Duration
	Namespace           string
	RollbackTimeout     time.Duration
	TargetNamespace     string
	UninstallTimeout    time.Duration
	UpgradeTimeout      time.Duration
	UserValuesConfigMap string
	UserValuesSecret    string
	Version             string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the Management Cluster the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the app to use for creating the repository directory structure.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the Workload Cluster belongs to.")
	cmd.Flags().BoolVar(&f.SkipMAPI, flagSkipMAPI, false, "Skip `mapi` directory when adding the app.")
	cmd.Flags().StringVar(&f.WorkloadCluster, flagWorkloadCluster, "", "Name of the Workload Cluster to configure the app for.")

	//CAPx only
	cmd.Flags().StringVar(&f.App, flagApp, "", "App name in the catalog.")
	cmd.Flags().StringVar(&f.Base, flagBase, "", "Path to the base directory. It must be relative to the repository root.")
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog to install the app from.")
	cmd.Flags().BoolVar(&f.InCluster, flagInCluster, false, "Use in-cluster configuration.")
	cmd.Flags().DurationVar(&f.InstallTimeout, flagInstallTimeout, 0, "Timeout for the Helm install.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace to install app into.")
	cmd.Flags().DurationVar(&f.RollbackTimeout, flagRollbackTimeout, 0, "Timeout for the Helm rollback.")
	cmd.Flags().StringVar(&f.TargetNamespace, flagTargetNamespace, "", "Namespace to install app into.")
	cmd.Flags().DurationVar(&f.UninstallTimeout, flagUninstallTimeout, 0, "Timeout for the Helm uninstall.")
	cmd.Flags().DurationVar(&f.UpgradeTimeout, flagUpgradeTimeout, 0, "Timeout for the Helm upgrade.")
	cmd.Flags().StringVar(&f.UserValuesConfigMap, flagUserValuesConfigMap, "", "Values YAML to customize the app with. Will get turn into a ConfigMap.")
	cmd.Flags().StringVar(&f.UserValuesSecret, flagUserValuesSecret, "", "Values YAML to customize the app with. Will get turn into a Secret.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to install.")

	_ = cmd.Flags().MarkDeprecated(flagNamespace, fmt.Sprintf("use %s instead", flagTargetNamespace))
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

	if f.Base == "" && f.App == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagApp)
	}

	if f.Base == "" && f.Catalog == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagCatalog)
	}

	if f.Base == "" && f.TargetNamespace == "" && f.Namespace == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagTargetNamespace)
	}

	if f.Base == "" && f.Version == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagVersion)
	}

	return nil
}
