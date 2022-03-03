package networkpool

import (
	"net"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	// Common.
	flagEnableLongNames = "enable-long-names"
	flagCIDRBlock       = "cidr-block"
	flagNetworkPoolName = "networkpool-name"
	flagOutput          = "output"
	flagOrganization    = "organization"
)

type flag struct {
	// Common.
	CIDRBlock       string
	EnableLongNames bool
	NetworkPoolName string
	Output          string
	Organization    string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.CIDRBlock, flagCIDRBlock, "", "Installation infrastructure provider.")
	cmd.Flags().BoolVar(&f.EnableLongNames, flagEnableLongNames, false, "Allow long names.")
	cmd.Flags().StringVar(&f.NetworkPoolName, flagNetworkPoolName, "", "NetworkPool identifier.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")

	_ = cmd.Flags().MarkHidden(flagEnableLongNames)
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

	if f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOrganization)
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
