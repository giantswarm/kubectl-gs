package autoupdate

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagManagementCluster = "management-cluster"
	flagOrganization      = "organization"
	flagWorkloadCluster   = "workload-cluster"

	// App CR part
	flagApp               = "app"
	flagRepository        = "repository"
	flagVersionRepository = "version-repository"
)

type flag struct {
	ManagementCluster string
	Name              string
	Organization      string
	WorkloadCluster   string

	// App CR part
	App               string
	Repository        string
	VersionRepository string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Codename of the Management Cluster the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Name of the Organization the Workload Cluster belongs to.")
	cmd.Flags().StringVar(&f.WorkloadCluster, flagWorkloadCluster, "", "Name of the Workload Cluster the app belongs to.")

	//App stuff
	cmd.Flags().StringVar(&f.App, flagApp, "", "App in the repository to configure automatic updates for.")
	cmd.Flags().StringVar(&f.Repository, flagRepository, "", "The gitops repository to update application in.")
	cmd.Flags().StringVar(&f.VersionRepository, flagVersionRepository, "", "The OCR repository to update the version from.")
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

	if f.Repository == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagRepository)
	}

	if f.VersionRepository == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagVersionRepository)
	}

	return nil
}
