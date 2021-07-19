package nodepool

import (
	"fmt"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	flagProvider = "provider"

	// AWS only.
	flagAWSInstanceType                     = "aws-instance-type"
	flagMachineDeploymentSubnet             = "machine-deployment-subnet"
	flagOnDemandBaseCapacity                = "on-demand-base-capacity"
	flagOnDemandPercentageAboveBaseCapacity = "on-demand-percentage-above-base-capacity"
	flagUseAlikeInstanceTypes               = "use-alike-instance-types"

	// Azure only.
	flagAzureVMSize       = "azure-vm-size"
	flagAzureUseSpotVMs   = "azure-spot-vms"
	flagAzureSpotMaxPrice = "azure-spot-max-price"

	// Common.
	flagAvailabilityZones = "availability-zones"
	flagClusterID         = "cluster-id"
	flagNodepoolName      = "nodepool-name"
	flagNodesMax          = "nodes-max"
	flagNodesMin          = "nodes-min"
	flagNodexMax          = "nodex-max"
	flagNodexMin          = "nodex-min"
	flagOutput            = "output"
	flagOwner             = "owner"
	flagRelease           = "release"
)

const (
	minNodes = 3
	maxNodes = 10
)

type flag struct {
	Provider string

	// AWS only.
	AWSInstanceType                     string
	MachineDeploymentSubnet             string
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	UseAlikeInstanceTypes               bool

	// Azure only.
	AzureVMSize       string
	AzureUseSpotVms   bool
	AzureSpotMaxPrice float32

	// Common.
	AvailabilityZones []string
	ClusterID         string
	NodepoolName      string
	NodesMax          int
	NodesMin          int
	Output            string
	Owner             string
	Release           string

	// Deprecated
	// Can be removed in a future version around March 2021 or later.
	NodexMin int
	NodexMax int
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Installation infrastructure provider.")

	// AWS only.
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge'.")
	cmd.Flags().StringVar(&f.MachineDeploymentSubnet, flagMachineDeploymentSubnet, "", "Subnet used for the Node Pool.")
	cmd.Flags().IntVar(&f.OnDemandBaseCapacity, flagOnDemandBaseCapacity, 0, "Number of base capacity for On demand instance distribution. Default is 0.")
	cmd.Flags().IntVar(&f.OnDemandPercentageAboveBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, 100, "Percentage above base capacity for On demand instance distribution. Default is 100.")
	cmd.Flags().BoolVar(&f.UseAlikeInstanceTypes, flagUseAlikeInstanceTypes, false, "Whether to use similar instances types as a fallback.")

	// Azure only.
	cmd.Flags().StringVar(&f.AzureVMSize, flagAzureVMSize, "Standard_D4s_v3", "Azure VM size to use for workers, e.g. 'Standard_D4s_v3'.")
	cmd.Flags().BoolVar(&f.AzureUseSpotVms, flagAzureUseSpotVMs, false, "Whether to use Spot VMs for this Node Pool.")
	cmd.Flags().Float32Var(&f.AzureSpotMaxPrice, flagAzureSpotMaxPrice, 0, "Max hourly price in USD to pay for one spot VM.")

	// Common.
	cmd.Flags().StringSliceVar(&f.AvailabilityZones, flagAvailabilityZones, []string{}, "List of availability zones to use, instead of setting a number. Use comma to separate values.")
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "Workload cluster ID.")
	cmd.Flags().StringVar(&f.NodepoolName, flagNodepoolName, "Unnamed node pool", "NodepoolName or purpose description of the node pool.")
	cmd.Flags().IntVar(&f.NodesMax, flagNodesMax, maxNodes, fmt.Sprintf("Maximum number of worker nodes for the node pool. (default %d)", maxNodes))
	cmd.Flags().IntVar(&f.NodesMin, flagNodesMin, minNodes, fmt.Sprintf("Minimum number of worker nodes for the node pool. (default %d)", minNodes))
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Workload cluster owner organization.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Workload cluster release. If not given, this remains empty to match the workload cluster version via the Management API.")

	// This can be removed in a future version around March 2021 or later.
	cmd.Flags().IntVar(&f.NodexMax, flagNodexMax, 0, "")
	cmd.Flags().IntVar(&f.NodexMin, flagNodexMin, 0, "")
	_ = cmd.Flags().MarkDeprecated(flagNodexMax, "")
	_ = cmd.Flags().MarkDeprecated(flagNodexMin, "")
}

func (f *flag) Validate() error {
	if f.Provider != key.ProviderAWS && f.Provider != key.ProviderAzure {
		return microerror.Maskf(invalidFlagError, "--%s must be either aws or azure", flagProvider)
	}

	{
		// Validate machine type.
		switch f.Provider {
		case key.ProviderAWS:
			if f.AWSInstanceType == "" {
				return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagAWSInstanceType)
			}
		case key.ProviderAzure:
			if f.AzureVMSize == "" {
				return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagAzureVMSize)
			}
		}
	}

	if f.MachineDeploymentSubnet != "" {
		matchedSubnet, err := regexp.MatchString("^20|21|22|23|24|25|26|27|28$", f.MachineDeploymentSubnet)
		if err == nil && !matchedSubnet {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid subnet size (20, 21, 22, 23, 24,25, 26, 27 or 28)", flagMachineDeploymentSubnet)
		}
	}

	if f.ClusterID == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagClusterID)
	}
	if f.NodepoolName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNodepoolName)
	}

	{
		if f.NodexMin > 0 {
			return microerror.Maskf(invalidFlagError, "please use --nodes-min instead of --nodex-min")
		}
		if f.NodexMax > 0 {
			return microerror.Maskf(invalidFlagError, "please use --nodes-max instead of --nodex-max")
		}

		// Validate scaling.
		if f.NodesMax < 0 {
			return microerror.Maskf(invalidFlagError, "--%s must be >= 0", flagNodesMax)
		}
		if f.NodesMin < 0 {
			return microerror.Maskf(invalidFlagError, "--%s must be >= 0", flagNodesMin)
		}
		if f.NodesMin > f.NodesMax {
			return microerror.Maskf(invalidFlagError, "--%s must be <= --%s", flagNodesMin, flagNodesMax)
		}
	}

	if f.Owner == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOwner)
	}

	{
		// Validate Availability Zones.
		var azs []string
		var numOfAZs int
		{
			if len(f.AvailabilityZones) > 0 {
				azs = f.AvailabilityZones
				numOfAZs = len(azs)
			}
		}

		// XXX: The availability zones can be set to nil on Azure.
		// https://github.com/giantswarm/giantswarm/issues/12860
		if f.Provider == key.ProviderAWS && numOfAZs < 1 {
			return microerror.Maskf(invalidFlagError, "--%s must be configured with at least 1 AZ", flagAvailabilityZones)
		}
	}

	{
		// Validate Spot instances.

		switch f.Provider {
		case key.ProviderAWS:
			if f.OnDemandBaseCapacity < 0 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagOnDemandBaseCapacity)
			}

			if f.OnDemandPercentageAboveBaseCapacity < 0 || f.OnDemandPercentageAboveBaseCapacity > 100 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 and lower than 100", flagOnDemandPercentageAboveBaseCapacity)
			}
		case key.ProviderAzure:
			if f.OnDemandBaseCapacity != 0 || f.OnDemandPercentageAboveBaseCapacity != 100 || f.UseAlikeInstanceTypes {
				return microerror.Maskf(invalidFlagError, "--%s, --%s and --%s spot instances flags are not supported on Azure.", flagOnDemandBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, flagUseAlikeInstanceTypes)
			}
		}
	}

	return nil
}
