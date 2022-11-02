package nodepool

import (
	"fmt"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

const (
	flagEnableLongNames = "enable-long-names"
	flagProvider        = "provider"

	// AWS only.
	flagAWSInstanceType                     = "aws-instance-type"
	flagMachineDeploymentSubnet             = "machine-deployment-subnet"
	flagOnDemandBaseCapacity                = "on-demand-base-capacity"
	flagOnDemandPercentageAboveBaseCapacity = "on-demand-percentage-above-base-capacity"
	flagUseAlikeInstanceTypes               = "use-alike-instance-types"
	flagEKS                                 = "aws-eks"
	flagClusterNamespace                    = "aws-cluster-namespace"

	// Azure only.
	flagAzureVMSize          = "azure-vm-size"
	flagAzureUseSpotVMs      = "azure-spot-vms"
	flagAzureSpotVMsMaxPrice = "azure-spot-vms-max-price"

	// Common.
	flagAvailabilityZones = "availability-zones"
	flagClusterName       = "cluster-name"
	flagDescription       = "description"
	flagNodesMax          = "nodes-max"
	flagNodesMin          = "nodes-min"
	flagOutput            = "output"
	flagOrganization      = "organization"
	flagRelease           = "release"
)

const (
	minNodes = 3
	maxNodes = 10
)

type flag struct {
	EnableLongNames bool
	Provider        string

	// AWS only.
	AWSInstanceType                     string
	MachineDeploymentSubnet             string
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	UseAlikeInstanceTypes               bool
	EKS                                 bool
	ClusterNamespace                    string

	// Azure only.
	AzureVMSize          string
	AzureUseSpotVms      bool
	AzureSpotVMsMaxPrice float32

	// Common.
	AvailabilityZones []string
	ClusterName       string
	Description       string
	NodesMax          int
	NodesMin          int
	Output            string
	Organization      string
	Release           string

	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.EnableLongNames, flagEnableLongNames, true, "Allow long names.")
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Installation infrastructure provider.")

	// AWS only.
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge'.")
	cmd.Flags().StringVar(&f.MachineDeploymentSubnet, flagMachineDeploymentSubnet, "", "Subnet used for the Node Pool.")
	cmd.Flags().IntVar(&f.OnDemandBaseCapacity, flagOnDemandBaseCapacity, 0, "Number of base capacity for On demand instance distribution. Default is 0. Only available on AWS.")
	cmd.Flags().IntVar(&f.OnDemandPercentageAboveBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, 100, "Percentage above base capacity for On demand instance distribution. Default is 100. Only available on AWS.")
	cmd.Flags().BoolVar(&f.UseAlikeInstanceTypes, flagUseAlikeInstanceTypes, false, "Whether to use similar instances types as a fallback. Only available on AWS.")
	cmd.Flags().BoolVar(&f.EKS, flagEKS, false, "Enable AWSEKS. Only available for AWS Release v20.0.0 (CAPA)")
	cmd.Flags().StringVar(&f.ClusterNamespace, flagClusterNamespace, "", "Namespace of the cluster to add the node pool to. Defaults to the organization namespace from v16.0.0 and to `default` before.")

	// Azure only.
	cmd.Flags().StringVar(&f.AzureVMSize, flagAzureVMSize, "Standard_D4s_v3", "Azure VM size to use for workers, e.g. 'Standard_D4s_v3'.")
	cmd.Flags().BoolVar(&f.AzureUseSpotVms, flagAzureUseSpotVMs, false, "Whether to use Spot VMs for this Node Pool. Defaults to false. Only available on Azure.")
	cmd.Flags().Float32Var(&f.AzureSpotVMsMaxPrice, flagAzureSpotVMsMaxPrice, 0, "Max hourly price in USD to pay for one spot VM on Azure. If not set, the on-demand price is used as the limit.")

	// Common.
	cmd.Flags().StringSliceVar(&f.AvailabilityZones, flagAvailabilityZones, []string{}, "List of availability zones to use, instead of setting a number. Use comma to separate values.")
	cmd.Flags().StringVar(&f.ClusterName, flagClusterName, "", "Name of the cluster to add the node pool to.")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "User-friendly description of the node pool's purpose.")
	cmd.Flags().IntVar(&f.NodesMax, flagNodesMax, maxNodes, fmt.Sprintf("Maximum number of worker nodes for the node pool. (default %d)", maxNodes))
	cmd.Flags().IntVar(&f.NodesMin, flagNodesMin, minNodes, fmt.Sprintf("Minimum number of worker nodes for the node pool. (default %d)", minNodes))
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Workload cluster release.")

	_ = cmd.Flags().MarkHidden(flagEnableLongNames)
	_ = cmd.Flags().MarkDeprecated(flagEnableLongNames, "Long names are supported by default, so this flag is not needed anymore and will be removed in the next major version.")

	// TODO: Make this flag visible when we roll CAPA/EKS out for customers
	_ = cmd.Flags().MarkHidden(flagEKS)

	f.print = genericclioptions.NewPrintFlags("")
	f.print.OutputFormat = nil

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
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

	if f.ClusterName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagClusterName)
	}
	if f.Description == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDescription)
	}

	{
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

	if f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOrganization)
	}

	if f.Release == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
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
