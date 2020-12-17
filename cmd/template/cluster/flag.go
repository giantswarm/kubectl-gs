package cluster

import (
	"net"
	"regexp"

	"github.com/mpvl/unique"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
)

const (
	flagProvider = "provider"

	// AWS only.
	flagExternalSNAT = "external-snat"
	flagPodsCIDR     = "pods-cidr"

	// Common.
	flagClusterID = "cluster-id"
	flagMasterAZ  = "master-az"
	flagName      = "name"
	flagOutput    = "output"
	flagOwner     = "owner"
	flagRelease   = "release"
	flagLabel     = "label"
)

type flag struct {
	Provider string

	// AWS only.
	ExternalSNAT bool
	PodsCIDR     string

	// Common.
	ClusterID string
	MasterAZ  []string
	Name      string
	Output    string
	Owner     string
	Release   string
	Label     []string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Installation infrastructure provider.")

	// AWS only.
	cmd.Flags().BoolVar(&f.ExternalSNAT, flagExternalSNAT, false, "AWS CNI configuration.")
	cmd.Flags().StringVar(&f.PodsCIDR, flagPodsCIDR, "", "CIDR used for the pods.")

	// Common.
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "User-defined cluster ID.")
	cmd.Flags().StringSliceVar(&f.MasterAZ, flagMasterAZ, []string{}, "Tenant master availability zone.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Tenant cluster name.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
	cmd.Flags().StringSliceVar(&f.Label, flagLabel, nil, "Tenant cluster label.")
}

func (f *flag) Validate() error {
	var err error

	if f.Provider != key.ProviderAWS && f.Provider != key.ProviderAzure {
		return microerror.Maskf(invalidFlagError, "--%s must be either aws or azure", flagProvider)
	}

	if f.ClusterID != "" {
		if len(f.ClusterID) != key.IDLength {
			return microerror.Maskf(invalidFlagError, "--%s must be length of %d", flagClusterID, key.IDLength)
		}

		matched, err := regexp.MatchString("^([a-z]+|[0-9]+)$", f.ClusterID)
		if err == nil && matched {
			// strings is letters only, which we also avoid
			return microerror.Maskf(invalidFlagError, "--%s must be alphanumeric", flagClusterID)
		}

		matched, err = regexp.MatchString("^[a-z0-9]+$", f.ClusterID)
		if err == nil && !matched {
			return microerror.Maskf(invalidFlagError, "--%s must only contain [a-z0-9]", flagClusterID)
		}

		return nil
	}
	// Validate name for non-aws clusters.
	if f.Provider != key.ProviderAWS && f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.PodsCIDR != "" {
		if !validateCIDR(f.PodsCIDR) {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagPodsCIDR)
		}
	}
	if f.Owner == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOwner)
	}

	{
		// Validate Master AZs.
		switch f.Provider {
		case key.ProviderAWS:
			if len(f.MasterAZ) != 0 && len(f.MasterAZ) != 1 && len(f.MasterAZ) != 3 {
				return microerror.Maskf(invalidFlagError, "--%s must be set to either one or three availability zone names", flagMasterAZ)
			}
			if !unique.StringsAreUnique(f.MasterAZ) {
				return microerror.Maskf(invalidFlagError, "--%s values must contain each AZ name only once", flagMasterAZ)
			}
		case key.ProviderAzure:
			if len(f.MasterAZ) > 1 {
				return microerror.Maskf(invalidFlagError, "--%s supports one availability zone only", flagMasterAZ)
			}
		}
	}

	// Validate release version for non-aws clusters.
	if f.Provider != key.ProviderAWS && f.Release == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
	}

	_, err = clusterlabels.Parse(f.Label)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagLabel, err)
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
