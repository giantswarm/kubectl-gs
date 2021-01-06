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
	flagOwner           = "owner"
)

type flag struct {
	// Common.
	CIDRBlock       string
	NetworkPoolName string
	Output          string
	Owner           string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.NetworkPoolName, flagNetworkPoolName, "", "NetworkPool identifier.")
	cmd.Flags().StringVar(&f.CIDRBlock, flagCIDRBlock, "", "Installation infrastructure provider.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Workload cluster owner organization.")
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
	if f.Owner == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOwner)
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
