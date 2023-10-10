package app

import (
	"fmt"
	"time"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v2/pkg/annotations"
	"github.com/giantswarm/kubectl-gs/v2/pkg/labels"
)

const (
	flagAppName                    = "app-name"
	flagCatalog                    = "catalog"
	flagCatalogNamespace           = "catalog-namespace"
	flagCluster                    = "cluster"
	flagClusterName                = "cluster-name"
	flagDefaultingEnabled          = "defaulting-enabled"
	flagInCluster                  = "in-cluster"
	flagInstallTimeout             = "install-timeout"
	flagName                       = "name"
	flagNamespace                  = "namespace"
	flagTargetNamespace            = "target-namespace"
	flagNamespaceConfigAnnotations = "namespace-annotations"
	flagNamespaceConfigLabels      = "namespace-labels"
	flagOrganization               = "organization"
	flagPreventDeletion            = "prevent-deletion"
	flagRollbackTimeout            = "rollback-timeout"
	flagUninstallTimeout           = "uninstall-timeout"
	flagUpgradeTimeout             = "upgrade-timeout"
	flagUserConfigMap              = "user-configmap"
	flagUserSecret                 = "user-secret"
	flagVersion                    = "version"
)

type flag struct {
	AppName                        string
	Catalog                        string
	CatalogNamespace               string
	Cluster                        string
	ClusterName                    string
	DefaultingEnabled              bool
	InCluster                      bool
	InstallTimeout                 time.Duration
	Name                           string
	Namespace                      string
	TargetNamespace                string
	Organization                   string
	RollbackTimeout                time.Duration
	Version                        string
	flagNamespaceConfigAnnotations []string
	flagNamespaceConfigLabels      []string
	UninstallTimeout               time.Duration
	UpgradeTimeout                 time.Duration
	flagUserConfigMap              string
	flagUserSecret                 string
	PreventDeletion                bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.AppName, flagAppName, "", "Optionally set a different name for the App CR.")
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog name where app is stored.")
	cmd.Flags().StringVar(&f.CatalogNamespace, flagCatalogNamespace, "", "Catalog namespace where Catalog CR is stored if outside the default namespace.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the app in the Catalog.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace where the app will be deployed")
	cmd.Flags().StringVar(&f.TargetNamespace, flagTargetNamespace, "", "Namespace where the app will be deployed.")
	cmd.Flags().StringVar(&f.Cluster, flagCluster, "", "Name of the cluster the app will be deployed to")
	cmd.Flags().StringVar(&f.ClusterName, flagClusterName, "", "Name of the cluster the app will be deployed to.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")
	cmd.Flags().BoolVar(&f.DefaultingEnabled, flagDefaultingEnabled, true, "Don't template fields that will be defaulted.")
	cmd.Flags().BoolVar(&f.InCluster, flagInCluster, false, fmt.Sprintf("Deploy the app in the current management cluster rather than in a workload cluster. If this is set, --%s will be ignored.", flagClusterName))
	cmd.Flags().DurationVar(&f.InstallTimeout, flagInstallTimeout, 0, "Timeout for the Helm install.")
	cmd.Flags().DurationVar(&f.RollbackTimeout, flagRollbackTimeout, 0, "Timeout for the Helm rollback.")
	cmd.Flags().DurationVar(&f.UninstallTimeout, flagUninstallTimeout, 0, "Timeout for the Helm uninstall.")
	cmd.Flags().DurationVar(&f.UpgradeTimeout, flagUpgradeTimeout, 0, "Timeout for the Helm upgrade.")
	cmd.Flags().StringVar(&f.flagUserConfigMap, flagUserConfigMap, "", "Path to the user values configmap YAML file.")
	cmd.Flags().StringVar(&f.flagUserSecret, flagUserSecret, "", "Path to the user secrets YAML file.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to be installed.")
	cmd.Flags().StringSliceVar(&f.flagNamespaceConfigAnnotations, flagNamespaceConfigAnnotations, nil, "Namespace configuration annotations in form key=value.")
	cmd.Flags().StringSliceVar(&f.flagNamespaceConfigLabels, flagNamespaceConfigLabels, nil, "Namespace configuration labels in form key=value.")
	cmd.Flags().BoolVar(&f.PreventDeletion, flagPreventDeletion, false, fmt.Sprintf("Label the App and other templated resources with '%s' to prevent deletion (see https://docs.giantswarm.io/advanced/deletion-prevention/).", label.PreventDeletion))

	_ = cmd.Flags().MarkDeprecated(flagNamespace, fmt.Sprintf("use --%s instead", flagTargetNamespace))
	_ = cmd.Flags().MarkDeprecated(flagCluster, fmt.Sprintf("use --%s instead", flagClusterName))

	_ = cmd.Flags().MarkHidden(flagOrganization)
}

func (f *flag) Validate() error {
	if f.Catalog == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCatalog)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.Namespace == "" && f.TargetNamespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagTargetNamespace)
	}
	if !f.InCluster && f.Cluster == "" && f.ClusterName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagClusterName)
	}
	if f.Version == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagVersion)
	}

	_, err := labels.Parse(f.flagNamespaceConfigLabels)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagNamespaceConfigLabels, err)
	}

	_, err = annotations.Parse(f.flagNamespaceConfigAnnotations)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid annotation definitions (%s)", flagNamespaceConfigAnnotations, err)
	}

	return nil
}
