package networkpool

import (
	"net"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	// Common.
	flagCIDRBlock       = "cidr-block"
	flagNetworkPoolName = "networkpool-name"
	flagOutput          = "output"
	flagOrganization    = "org"
	flagOwner           = "owner" // TODO: Remove some time after December 2021
)

type flag struct {
	// Common.
	CIDRBlock       string
	NetworkPoolName string
	Output          string
	Organization    string
	Owner           string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.NetworkPoolName, flagNetworkPoolName, "", "NetworkPool identifier.")
	cmd.Flags().StringVar(&f.CIDRBlock, flagCIDRBlock, "", "Installation infrastructure provider.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Workload cluster owner organization.")

	// TODO: Remove around December 2021
	_ = cmd.Flags().MarkDeprecated(flagOwner, "please use --org instead.")
}

func (f *flag) Validate() error {
	if f.NetworkPoolName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNetworkPoolName)
	}
	if f.CIDRBlock == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCIDRBlock)
	}
	if f.CIDRBlock != "" {
		if !validateCIDR(f.CIDRBlock) {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagCIDRBlock)
		}
	}

	// TODO: Remove the flag completely some time after December 2021
	if f.Owner == "" && f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s or --%s must not be empty", flagOwner, flagOrganization)
	}

	if f.Owner != "" {
		if f.Organization != "" {
			return microerror.Maskf(invalidFlagError, "--%s and --%s cannot be combined", flagOwner, flagOrganization)
		}

		f.Organization = f.Owner
		f.Owner = ""
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
