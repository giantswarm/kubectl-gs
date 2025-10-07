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
	flagSkipMAPI          = "skip-mapi"
	flagRepositoryName    = "repository-name"

	// CAPI only
	flagRelease           = "release"
	flagClusterUserConfig = "cluster-user-config"
)

type flag struct {
	Base              string
	ManagementCluster string
	Name              string
	Organization      string
	SkipMAPI          bool
	RepositoryName    string

	Release           string
	ClusterUserConfig string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Base, flagBase, "", "Path to the base directory. It must be relative to the repository root.")
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the management cluster the workload cluster belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the Workload Cluster.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the workload cluster belongs to.")
	cmd.Flags().BoolVar(&f.SkipMAPI, flagSkipMAPI, false, "Skip `mapi` directory when adding the app.")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the GitOps repository.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Workload cluster release version.")
	cmd.Flags().StringVar(&f.ClusterUserConfig, flagClusterUserConfig, "", "Cluster app user configuration to patch the base with.")
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

	if f.Base != "" && f.Release == "" {
		return microerror.Maskf(
			invalidFlagsError,
			"--%s must not be empty when referencing base",
			flagRelease,
		)
	}

	return nil
}
