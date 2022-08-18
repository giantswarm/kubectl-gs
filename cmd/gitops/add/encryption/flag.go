package encryption

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagFingerprint       = "fingerprint"
	flagGenerate          = "generate"
	flagManagementCluster = "management-cluster"
	flagOrganization      = "organization"
	flagTarget            = "target"
	flagWorkloadCluster   = "workload-cluster"
)

type flag struct {
	Fingerprint       string
	Generate          bool
	ManagementCluster string
	Organization      string
	Target            string
	WorkloadCluster   string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Fingerprint, flagFingerprint, "", "Fingerprint of the key pair to configure repository with.")
	cmd.Flags().BoolVar(&f.Generate, flagGenerate, false, "Generate new key pair.")
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Management cluster to configure the encryption for.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Organization in the Management Cluster to configure the encryption for.")
	cmd.Flags().StringVar(&f.Target, flagTarget, "secrets/", "Relative directory to configure the encryption for.")
	cmd.Flags().StringVar(&f.WorkloadCluster, flagWorkloadCluster, "", "Workload Cluster in the Organization to configure the encryption for.")
}

func (f *flag) Validate() error {
	if !f.Generate && f.Fingerprint == "" {
		return microerror.Maskf(invalidFlagsError, "either one of the, --%s or %s, must be specified", flagGenerate, flagFingerprint)
	}

	if f.ManagementCluster == "" {
		return microerror.Maskf(invalidFlagsError, "at least the --%s must be specified", flagManagementCluster)
	}

	if f.WorkloadCluster != "" && f.Organization == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must be specified when --%s is used", flagOrganization, flagWorkloadCluster)
	}

	if f.Target == "/" && f.WorkloadCluster == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be set to '/' when used with Management Cluster or Organization only", flagTarget)
	}

	return nil
}
