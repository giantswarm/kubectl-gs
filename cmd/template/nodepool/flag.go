package nodepool

import (
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/aws"
	"github.com/giantswarm/kubectl-gs/pkg/gsrelease"
)

const (
	flagAvailabilityZones                   = "availability-zones"
	flagAWSInstanceType                     = "aws-instance-type"
	flagClusterID                           = "cluster-id"
	flagNodepoolName                        = "nodepool-name"
	flagNoCache                             = "no-cache"
	flagNodesMax                            = "nodex-max"
	flagNodesMin                            = "nodex-min"
	flagNumAvailabilityZones                = "num-availability-zones"
	flagOnDemandBaseCapacity                = "on-demand-base-capacity"
	flagOnDemandPercentageAboveBaseCapacity = "on-demand-precentage-above-base-capacity"
	flagOutput                              = "output"
	flagOwner                               = "owner"
	flagRegion                              = "region"
	flagRelease                             = "release"
)

type flag struct {
	AvailabilityZones                   string
	AWSInstanceType                     string
	ClusterID                           string
	NodepoolName                        string
	NoCache                             bool
	NodesMax                            int
	NodesMin                            int
	NumAvailabilityZones                int
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	Output                              string
	Owner                               string
	Region                              string
	Release                             string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.AvailabilityZones, flagAvailabilityZones, "", "List of availability zones to use, instead of setting a number. Use comma to separate values.")
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge'.")
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "Tenant cluster ID.")
	cmd.Flags().StringVar(&f.NodepoolName, flagNodepoolName, "Unnamed node pool", "NodepoolName or purpose description of the node pool.")
	cmd.Flags().BoolVar(&f.NoCache, flagNoCache, false, "Force updating release folder.")
	cmd.Flags().IntVar(&f.NodesMax, flagNodesMax, 10, "Maximum number of worker nodes for the node pool.")
	cmd.Flags().IntVar(&f.NodesMin, flagNodesMin, 3, "Minimum number of worker nodes for the node pool.")
	cmd.Flags().IntVar(&f.NumAvailabilityZones, flagNumAvailabilityZones, 1, "Number of availability zones to use. Default is 1.")
	cmd.Flags().IntVar(&f.OnDemandBaseCapacity, flagOnDemandBaseCapacity, 0, "Number of base capacity for On demand instance distribution. Default is 0.")
	cmd.Flags().IntVar(&f.OnDemandPercentageAboveBaseCapacity, flagOnDemandPercentageAboveBaseCapacity, 100, "Percentage above base capacity for On demand instance distribution. Default is 100.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs. (default: stdout)")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region (e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
}

func (f *flag) Validate() error {
	var err error

	if f.AWSInstanceType == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagAWSInstanceType)
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
	if f.Region == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRegion)
	}
	if !aws.ValidateRegion(f.Region) {
		return microerror.Maskf(invalidFlagError, "--%s must be valid region name", flagRegion)
	}

	if f.AvailabilityZones != "" {
		azs := strings.Split(f.AvailabilityZones, ",")
		if len(azs) < 1 {
			return microerror.Maskf(invalidFlagError, "--%s must be configured with at least 1 AZ", flagAvailabilityZones)
		}
		if len(azs) > aws.AvailableAZs(f.Region) {
			return microerror.Maskf(invalidFlagError, "--%s must be less than number of available AZs in selected region)", flagAvailabilityZones)
		}
		for _, az := range azs {
			if !aws.ValidateAZ(f.Region, az) {
				return microerror.Maskf(invalidFlagError, "--%s must be a list with valid AZs for selected region", flagAvailabilityZones)

			}
		}
	} else {
		if f.NumAvailabilityZones < 1 {
			if f.AvailabilityZones == "" {
				return microerror.Maskf(invalidFlagError, "--%s must be > 1 when --%s not specified)", flagNumAvailabilityZones, flagAvailabilityZones)
			}
			if f.NumAvailabilityZones > aws.AvailableAZs(f.Region) {
				return microerror.Maskf(invalidFlagError, "--%s must be less than number of available AZs in selected region)", flagNumAvailabilityZones)
			}
		}
	}

	if f.OnDemandBaseCapacity < 0 {
		return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagOnDemandBaseCapacity)
	}

	if f.OnDemandPercentageAboveBaseCapacity < 0 || f.OnDemandPercentageAboveBaseCapacity > 100 {
		return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 and lower than 100", flagOnDemandPercentageAboveBaseCapacity)
	}

	if f.Release == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
	}

	var release *gsrelease.GSRelease
	{
		c := gsrelease.Config{
			NoCache: f.NoCache,
		}

		release, err = gsrelease.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if !release.Validate(f.Release) {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid release", flagRelease)
	}

	return nil
}
