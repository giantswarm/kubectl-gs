package autoupdate

import (
	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagManagementCluster = "management-cluster"
	flagOrganization      = "organization"
	flagSkipMAPI          = "skip-mapi"
	flagWorkloadCluster   = "workload-cluster"

	flagApp          = "app"
	flagVersionRange = "version-range"
)

type flag struct {
	ManagementCluster string
	Name              string
	Organization      string
	SkipMAPI          bool
	WorkloadCluster   string

	App          string
	VersionRange string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the Management Cluster the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the Workload Cluster belongs to.")
	cmd.Flags().BoolVar(&f.SkipMAPI, flagSkipMAPI, false, "Skip `mapi` directory when adding the app.")
	cmd.Flags().StringVar(&f.WorkloadCluster, flagWorkloadCluster, "", "Name of the Workload Cluster the app belongs to.")

	cmd.Flags().StringVar(&f.App, flagApp, "", "App in the repository to configure automatic updates for.")
	cmd.Flags().StringVar(&f.VersionRange, flagVersionRange, ">=0.0.0", "Semver range of the chart versions to update the app to.")
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

	if f.VersionRange == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagVersionRange)
	}

	_, err := semver.NewConstraint(f.VersionRange)
	if err != nil {
		return microerror.Maskf(invalidFlagsError, "--%s %q is not a semver range: %s", flagVersionRange, f.VersionRange, err)
	}

	return nil
}
