package nodepool

import (
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/release"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/aws"
)

const (
	flagProvider = "provider"

	// AWS only.
	flagAWSInstanceType                     = "aws-instance-type"
	flagOnDemandBaseCapacity                = "on-demand-base-capacity"
	flagOnDemandPercentageAboveBaseCapacity = "on-demand-percentage-above-base-capacity"
	flagUseAlikeInstanceTypes               = "use-alike-instance-types"

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
	cmd.Flags().StringVar(&f.Provider, flagProvider, key.ProviderAWS, "Installation infrastructure provider (e.g. aws or azure).")

	// AWS only.
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge'.")
	cmd.Flags().IntVar(&f.OnDemandBaseCapacity, flagOnDemandBaseCapacity, 0, "Number of base capacity for On demand instance distribution. Default is 0.")
	cmd.Flags().IntVar(&f.OnDemandPercentageAboveBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, 100, "Percentage above base capacity for On demand instance distribution. Default is 100.")
	cmd.Flags().BoolVar(&f.UseAlikeInstanceTypes, flagUseAlikeInstanceTypes, false, "Whether to use similar instances types as a fallback.")

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

		if numOfAZs < 1 {
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
		}
	}

	{
		// Spot instances.
		if f.Provider == key.ProviderAWS {
			if f.OnDemandBaseCapacity < 0 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagOnDemandBaseCapacity)
			}

			if f.OnDemandPercentageAboveBaseCapacity < 0 || f.OnDemandPercentageAboveBaseCapacity > 100 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 and lower than 100", flagOnDemandPercentageAboveBaseCapacity)
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
