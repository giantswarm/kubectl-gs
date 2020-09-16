package nodepool

import (
	"encoding/base64"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/azure"
	"github.com/giantswarm/kubectl-gs/pkg/release"

	"github.com/giantswarm/kubectl-gs/pkg/aws"
)

const (
	flagProvider = "provider"

	// AWS only.
	flagAWSInstanceType                     = "aws-instance-type"
	flagOnDemandBaseCapacity                = "on-demand-base-capacity"
	flagOnDemandPercentageAboveBaseCapacity = "on-demand-percentage-above-base-capacity"
	flagUseAlikeInstanceTypes               = "use-alike-instance-types"

	// Azure only.
	flagPublicSSHKey = "public-ssh-key"
	flagAzureVMSize  = "azure-vm-size"

	// Common.
	flagAvailabilityZones    = "availability-zones"
	flagClusterID            = "cluster-id"
	flagNodepoolName         = "nodepool-name"
	flagNodesMax             = "nodex-max"
	flagNodesMin             = "nodex-min"
	flagNumAvailabilityZones = "num-availability-zones"
	flagOutput               = "output"
	flagOwner                = "owner"
	flagRegion               = "region"
	flagRelease              = "release"
)

type flag struct {
	Provider string

	// AWS only.
	AWSInstanceType                     string
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	UseAlikeInstanceTypes               bool

	// Azure only.
	PublicSSHKey string
	AzureVMSize  string

	// Common.
	AvailabilityZones    []string
	ClusterID            string
	NodepoolName         string
	NodesMax             int
	NodesMin             int
	NumAvailabilityZones int
	Output               string
	Owner                string
	Region               string
	Release              string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Provider, flagProvider, key.ProviderAWS, "Installation infrastructure provider.")

	// AWS only.
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge'.")
	cmd.Flags().IntVar(&f.OnDemandBaseCapacity, flagOnDemandBaseCapacity, 0, "Number of base capacity for On demand instance distribution. Default is 0.")
	cmd.Flags().IntVar(&f.OnDemandPercentageAboveBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, 100, "Percentage above base capacity for On demand instance distribution. Default is 100.")
	cmd.Flags().BoolVar(&f.UseAlikeInstanceTypes, flagUseAlikeInstanceTypes, false, "Whether to use similar instances types as a fallback.")

	// Azure only.
	cmd.Flags().StringVar(&f.PublicSSHKey, flagPublicSSHKey, "", "Base64-encoded Azure machine public SSH key.")
	cmd.Flags().StringVar(&f.AzureVMSize, flagAzureVMSize, "Standard_D4_v3", "Azure VM size to use for workers, e.g. 'Standard_D4_v3'.")

	// Common.
	cmd.Flags().StringSliceVar(&f.AvailabilityZones, flagAvailabilityZones, []string{}, "List of availability zones to use, instead of setting a number. Use comma to separate values.")
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "Tenant cluster ID.")
	cmd.Flags().StringVar(&f.NodepoolName, flagNodepoolName, "Unnamed node pool", "NodepoolName or purpose description of the node pool.")
	cmd.Flags().IntVar(&f.NodesMax, flagNodesMax, 10, "Maximum number of worker nodes for the node pool.")
	cmd.Flags().IntVar(&f.NodesMin, flagNodesMin, 3, "Minimum number of worker nodes for the node pool.")
	cmd.Flags().IntVar(&f.NumAvailabilityZones, flagNumAvailabilityZones, 1, "Number of availability zones to use. Default is 1.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region (e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
}

func (f *flag) Validate() error {
	var err error

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

	if f.ClusterID == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagClusterID)
	}
	if f.NodepoolName == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNodepoolName)
	}
	if f.NodesMax < 1 {
		return microerror.Maskf(invalidFlagError, "--%s must be > 0", flagNodesMax)
	}
	if f.NodesMin < 1 {
		return microerror.Maskf(invalidFlagError, "--%s must be > 0", flagNodesMin)
	}
	if f.NodesMin > f.NodesMax {
		return microerror.Maskf(invalidFlagError, "--%s must be <= --%s", flagNodesMin, flagNodesMax)
	}

	if f.Owner == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOwner)
	}

	{
		// Validate installation region.
		if f.Region == "" {
			return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRegion)
		}

		switch f.Provider {
		case key.ProviderAWS:
			if !aws.ValidateRegion(f.Region) {
				return microerror.Maskf(invalidFlagError, "--%s must be valid region name", flagRegion)
			}
		case key.ProviderAzure:
			if !azure.ValidateRegion(f.Region) {
				return microerror.Maskf(invalidFlagError, "--%s must be valid region name", flagRegion)
			}
		}
	}

	{
		if f.Provider == key.ProviderAzure {
			if len(f.PublicSSHKey) < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must not be empty on Azure", flagPublicSSHKey)
			} else {
				sshKey := []byte(f.PublicSSHKey)
				dest := make([]byte, base64.StdEncoding.EncodedLen(len(sshKey)))
				_, err = base64.StdEncoding.Decode(dest, sshKey)
				if err != nil {
					return microerror.Maskf(invalidFlagError, "--%s must be Base64-encoded", flagPublicSSHKey)
				}
			}
		}
	}

	{
		// Validate Availability Zones.
		var azs []string
		var numOfAZs int
		{
			if len(f.AvailabilityZones) > 0 {
				azs = f.AvailabilityZones
				numOfAZs = len(azs)
			} else {
				numOfAZs = f.NumAvailabilityZones
			}
		}

		var numOfAvailableAZs int
		{
			switch f.Provider {
			case key.ProviderAWS:
				numOfAvailableAZs = aws.AvailableAZs(f.Region)
			}
		}

		// XXX: The availability zones can be set to nil on Azure.
		// https://github.com/giantswarm/giantswarm/issues/12860
		if f.Provider == key.ProviderAWS && numOfAZs < 1 {
			return microerror.Maskf(invalidFlagError, "--%s must be configured with at least 1 AZ", flagAvailabilityZones)
		}
		if numOfAZs > numOfAvailableAZs {
			return microerror.Maskf(invalidFlagError, "--%s must be less than number of available AZs in selected region)", flagAvailabilityZones)
		}

		switch f.Provider {
		case key.ProviderAWS:
			for _, az := range azs {
				if !aws.ValidateAZ(f.Region, az) {
					return microerror.Maskf(invalidFlagError, "--%s must be a list with valid AZs for selected region", flagAvailabilityZones)
				}
			}
		case key.ProviderAzure:
			for _, az := range azs {
				if !azure.ValidateAZ(f.Region, az) {
					return microerror.Maskf(invalidFlagError, "--%s must be a list with valid AZs for selected region", flagAvailabilityZones)
				}
			}
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

	{
		if f.Release == "" {
			return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
		}

		var r *release.Release
		{
			c := release.Config{}

			r, err = release.New(c)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		if !r.Validate(f.Release) {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid release", flagRelease)
		}
	}

	return nil
}
