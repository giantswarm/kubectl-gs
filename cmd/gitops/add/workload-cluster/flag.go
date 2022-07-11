package wcluster

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagBase              = "base"
	flagManagementCluster = "management-cluster"
	flagName              = "name"
	flagOrganization      = "organization"
	flagRepositoryName    = "repository-name"

	//CAPx only
	flagClusterRelease        = "cluster-release"
	flagClusterUserConfig     = "cluster-user-config"
	flagDefaultAppsRelease    = "default-apps-release"
	flagDefaultAppsUserConfig = "default-apps-user-config"
)

type flag struct {
	Base              string
	ManagementCluster string
	Name              string
	Organization      string
	RepositoryName    string

	//CAPx only
	ClusterRelease        string
	ClusterUserConfig     string
	DefaultAppsRelease    string
	DefaultAppsUserConfig string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Base, flagBase, "", "Path to the base directory. It must be relative to the repository root.")
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the Management Cluster the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the GitOps repository.")

	//CAPx only
	cmd.Flags().StringVar(&f.ClusterRelease, flagClusterRelease, "", "Cluster app version.")
	cmd.Flags().StringVar(&f.ClusterUserConfig, flagClusterUserConfig, "", "Cluster app user configuration to patch the base with.")
	cmd.Flags().StringVar(&f.DefaultAppsRelease, flagDefaultAppsRelease, "", "Default apps app version.")
	cmd.Flags().StringVar(&f.DefaultAppsUserConfig, flagDefaultAppsUserConfig, "", "Default apps app user configuration to patch the base with.")
}

func (f *flag) Validate() error {
	if f.ManagementCluster == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagManagementCluster)
	}

	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	if f.Organization == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagOrganization)
	}

	if f.RepositoryName == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagRepositoryName)
	}

	if f.Base != "" && (f.ClusterRelease == "" || f.DefaultAppsRelease == "") {
		return microerror.Maskf(
			invalidFlagsError,
			"both --%s and --%s must not be empty when referencing base",
			flagClusterRelease,
			flagDefaultAppsRelease,
		)
	}

	return nil
}
