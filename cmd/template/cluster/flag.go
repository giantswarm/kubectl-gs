package cluster

import (
	"net"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/labels"
)

const (
	flagProvider = "provider"

	// AWS only.
	flagExternalSNAT       = "external-snat"
	flagPodsCIDR           = "pods-cidr"
	flagControlPlaneSubnet = "control-plane-subnet"

	// Common.
	flagClusterIDDeprecated = "cluster-id"
	flagControlPlaneAZ      = "control-plane-az"
	flagDescription         = "description"
	flagMasterAZ            = "master-az" // TODO: Remove some time after August 2021
	flagName                = "name"
	flagOutput              = "output"
	flagOwner               = "owner"
	flagRelease             = "release"
	flagLabel               = "label"
)

type flag struct {
	Provider string

	// AWS only.
	ControlPlaneSubnet string
	ExternalSNAT       bool
	PodsCIDR           string

	// Common.
	ClusterIDDeprecated string
	ControlPlaneAZ      []string
	Description         string
	MasterAZ            []string
	Name                string
	Output              string
	Owner               string
	Release             string
	Label               []string

	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Installation infrastructure provider.")

	// AWS only.
	cmd.Flags().StringVar(&f.ControlPlaneSubnet, flagControlPlaneSubnet, "", "Subnet used for the Control Plane.")
	cmd.Flags().BoolVar(&f.ExternalSNAT, flagExternalSNAT, false, "AWS CNI configuration.")
	cmd.Flags().StringVar(&f.PodsCIDR, flagPodsCIDR, "", "CIDR used for the pods.")

	// Common.
	cmd.Flags().StringVar(&f.ClusterIDDeprecated, flagClusterIDDeprecated, "", "Unique identifier of the cluster (deprecated).")
	cmd.Flags().StringSliceVar(&f.ControlPlaneAZ, flagControlPlaneAZ, nil, "Availability zone(s) to use by control plane nodes.")
	cmd.Flags().StringSliceVar(&f.MasterAZ, flagMasterAZ, nil, "Replaced by --control-plane-az.")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "User-friendly description of the cluster's purpose (formerly called name).")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Unique identifier of the cluster (formerly called ID).")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Workload cluster owner organization.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Workload cluster release. If not given, this remains empty for defaulting to the most recent one via the Management API.")
	cmd.Flags().StringSliceVar(&f.Label, flagLabel, nil, "Workload cluster label.")

	// TODO: Remove the flag completely some time after August 2021
	_ = cmd.Flags().MarkDeprecated(flagMasterAZ, "please use --control-plane-az.")

	// TODO: Remove around December 2021
	_ = cmd.Flags().MarkDeprecated(flagClusterIDDeprecated, "please use --name instead.")

	f.config = genericclioptions.NewConfigFlags(true)
	f.print = genericclioptions.NewPrintFlags("")
	f.print.OutputFormat = nil

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.Flags())
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	var err error

	// TODO: Remove the flag completely some time after August 2021
	if len(f.MasterAZ) > 0 {
		if len(f.ControlPlaneAZ) > 0 {
			return microerror.Maskf(invalidFlagError, "--control-plane-az and --master-az cannot be combined")
		}

		f.ControlPlaneAZ = f.MasterAZ
		f.MasterAZ = nil
	}

	// Handle legacy cluster ID, pass it to cluster name flag.
	// TODO: Remove around December 2021
	if f.ClusterIDDeprecated != "" {
		f.Name = f.ClusterIDDeprecated
		f.ClusterIDDeprecated = ""
	}

	if f.Provider != key.ProviderAWS && f.Provider != key.ProviderAzure {
		return microerror.Maskf(invalidFlagError, "--%s must be either aws or azure", flagProvider)
	}

	if f.Name != "" {
		if len(f.Name) != key.IDLength {
			return microerror.Maskf(invalidFlagError, "--%s must be length of %d", flagClusterIDDeprecated, key.IDLength)
		}

		matchedLettersOnly, err := regexp.MatchString("^[a-z]+$", f.Name)
		if err == nil && matchedLettersOnly {
			// strings is letters only, which we avoid
			return microerror.Maskf(invalidFlagError, "--%s must contain at least one number", flagClusterIDDeprecated)
		}

		matchedNumbersOnly, err := regexp.MatchString("^[0-9]+$", f.Name)
		if err == nil && matchedNumbersOnly {
			// strings is numbers only, which we avoid
			return microerror.Maskf(invalidFlagError, "--%s must contain at least one digit", flagClusterIDDeprecated)
		}

		matched, err := regexp.MatchString("^[a-z][a-z0-9]+$", f.Name)
		if err == nil && !matched {
			return microerror.Maskf(invalidFlagError, "--%s must only contain alphanumeric characters, and start with a letter", flagClusterIDDeprecated)
		}

		if f.ControlPlaneSubnet != "" {
			matchedSubnet, err := regexp.MatchString("^20|21|22|23|24|25$", f.ControlPlaneSubnet)
			if err == nil && !matchedSubnet {
				return microerror.Maskf(invalidFlagError, "--%s must be a valid subnet size (20, 21, 22, 23, 24 or 25)", flagControlPlaneSubnet)
			}
		}

		return nil
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
			if len(f.ControlPlaneAZ) != 0 && len(f.ControlPlaneAZ) != 1 && len(f.ControlPlaneAZ) != 3 {
				return microerror.Maskf(invalidFlagError, "--%s must be set to either one or three availability zone names", flagControlPlaneAZ)
			}
		case key.ProviderAzure:
			if len(f.ControlPlaneAZ) > 1 {
				return microerror.Maskf(invalidFlagError, "--%s supports one availability zone only", flagControlPlaneAZ)
			}
		}
	}

	// Validate release version for non-aws clusters.
	if f.Provider != key.ProviderAWS && f.Release == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
	}

	_, err = labels.Parse(f.Label)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagLabel, err)
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
