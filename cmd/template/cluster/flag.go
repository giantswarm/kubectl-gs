package cluster

import (
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/aws"
	"github.com/giantswarm/kubectl-gs/pkg/gsrelease"
)

const (
	flagDomain                  = "domain"
	flagMasterAZ                = "master-az"
	flagName                    = "name"
	flagNoCache                 = "no-cache"
	flagOutput                  = "output"
	flagOwner                   = "owner"
	flagRegion                  = "region"
	flagRelease                 = "release"
	flagTemplateDefaultNodepool = "template-default-nodepool"

	// nodepool flags
	flagAvailabilityZones    = "availability-zones"
	flagAWSInstanceType      = "aws-instance-type"
	flagNodepoolName         = "nodepool-name"
	flagNodesMax             = "nodex-max"
	flagNodesMin             = "nodex-min"
	flagNumAvailabilityZones = "num-availability-zones"
)

type flag struct {
	Domain                  string
	MasterAZ                string
	Name                    string
	NoCache                 bool
	Output                  string
	Owner                   string
	Region                  string
	Release                 string
	TemplateDefaultNodepool bool

	// nodepool fields
	AvailabilityZones    string
	AWSInstanceType      string
	NodepoolName         string
	NodesMax             int
	NodesMin             int
	NumAvailabilityZones int
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Domain, flagDomain, "", "Installation base domain.")
	cmd.Flags().StringVar(&f.MasterAZ, flagMasterAZ, "", "Tenant master availability zone.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Tenant cluster name.")
	cmd.Flags().BoolVar(&f.NoCache, flagNoCache, false, "Force updating release folder.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region(e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
	cmd.Flags().BoolVar(&f.TemplateDefaultNodepool, flagTemplateDefaultNodepool, false, "Template default nodepool CRs with cluster CRs.")

	// nodepool validation
	// required only when template-default-nodepool
	cmd.Flags().StringVar(&f.AvailabilityZones, flagAvailabilityZones, "", "List of availability zones to use, instead of setting a number. Use comma to separate values (when --template-default-nodepool=true).")
	cmd.Flags().StringVar(&f.AWSInstanceType, flagAWSInstanceType, "m5.xlarge", "EC2 instance type to use for workers, e. g. 'm5.2xlarge' (when --template-default-nodepool=true).")
	cmd.Flags().StringVar(&f.NodepoolName, flagNodepoolName, "Unnamed node pool", "NodepoolName or purpose description of the node pool (when --template-default-nodepool=true).")
	cmd.Flags().IntVar(&f.NodesMax, flagNodesMax, 10, "Maximum number of worker nodes for the node pool (when --template-default-nodepool=true).")
	cmd.Flags().IntVar(&f.NodesMin, flagNodesMin, 3, "Minimum number of worker nodes for the node pool (when --template-default-nodepool=true).")
	cmd.Flags().IntVar(&f.NumAvailabilityZones, flagNumAvailabilityZones, 1, "Number of availability zones to use. Default is 1 (when --template-default-nodepool=true).")
}

func (f *flag) Validate() error {
	var err error

	if f.Domain == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDomain)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
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
	if f.MasterAZ == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagMasterAZ)
	}
	if !aws.ValidateAZ(f.Region, f.MasterAZ) {
		return microerror.Maskf(invalidFlagError, "--%s must be valid AZ name for selected region %s", flagMasterAZ, f.Region)
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

	if f.TemplateDefaultNodepool {
		if f.AWSInstanceType == "" {
			return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagAWSInstanceType)
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
	}

	return nil
}
