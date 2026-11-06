package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	clustercommon "github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
)

const (
	flagManagementCluster = "management-cluster"
	flagName              = "name"
	flagOrganization      = "organization"
	flagSkipMAPI          = "skip-mapi"
	flagWorkloadCluster   = "workload-cluster"

	// HelmRelease part
	flagApp                 = "app"
	flagBase                = "base"
	flagInCluster           = "in-cluster"
	flagInstallTimeout      = "install-timeout"
	flagNamespace           = "namespace"
	flagOCIURLPrefix        = "oci-url-prefix"
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

	// HelmRelease part
	App                 string
	Base                string
	InCluster           bool
	InstallTimeout      time.Duration
	Namespace           string
	OCIURLPrefix        string
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
	cmd.Flags().StringVar(&f.App, flagApp, "", "Name of the app's chart.")
	cmd.Flags().StringVar(&f.Base, flagBase, "", "Path to the base directory. It must be relative to the repository root.")
	cmd.Flags().BoolVar(&f.InCluster, flagInCluster, false, "Install the app in the management cluster instead of the workload cluster.")
	cmd.Flags().DurationVar(&f.InstallTimeout, flagInstallTimeout, 0, "Timeout for the Helm install.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace to install app into.")
	cmd.Flags().StringVar(&f.OCIURLPrefix, flagOCIURLPrefix, fmt.Sprintf("oci://%s/%s", clustercommon.GSOCIRegistry, strings.TrimSuffix(clustercommon.GSOCIChartsRepoPrefix, "/")), "OCI repository holding the chart. The chart URL is this prefix and --app.")
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

	if f.Base == "" && !strings.HasPrefix(f.OCIURLPrefix, "oci://") {
		return microerror.Maskf(invalidFlagsError, "--%s must start with oci://", flagOCIURLPrefix)
	}

	if f.Base == "" && f.TargetNamespace == "" && f.Namespace == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagTargetNamespace)
	}

	if f.Base == "" && f.Version == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagVersion)
	}

	return nil
}
