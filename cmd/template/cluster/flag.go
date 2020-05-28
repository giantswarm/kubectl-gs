package cluster

import (
	"net"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/mpvl/unique"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/aws"
	"github.com/giantswarm/kubectl-gs/pkg/gsrelease"
)

const (
	flagClusterID    = "cluster-id"
	flagDomain       = "domain"
	flagMasterAZ     = "master-az"
	flagName         = "name"
	flagNoCache      = "no-cache"
	flagPodsCIDR     = "pods-cidr"
	flagExternalSNAT = "external-snat"
	flagOutput       = "output"
	flagOwner        = "owner"
	flagRegion       = "region"
	flagRelease      = "release"
)

type flag struct {
	ClusterID    string
	Domain       string
	MasterAZ     []string
	Name         string
	NoCache      bool
	PodsCIDR     string
	ExternalSNAT bool
	Output       string
	Owner        string
	Region       string
	Release      string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Domain, flagDomain, "", "Installation base domain.")
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "User-defined cluster ID.")
	cmd.Flags().BoolVar(&f.ExternalSNAT, flagExternalSNAT, false, "AWS CNI configuration.")
	cmd.Flags().StringSliceVar(&f.MasterAZ, flagMasterAZ, []string{}, "Tenant master availability zone.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Tenant cluster name.")
	cmd.Flags().BoolVar(&f.NoCache, flagNoCache, false, "Force updating release folder.")
	cmd.Flags().StringVar(&f.PodsCIDR, flagPodsCIDR, "", "CIDR used for the pods.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region (e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
}

func (f *flag) Validate() error {
	var err error

	if f.ClusterID != "" {
		if len(f.ClusterID) != key.IDLength {
			return microerror.Maskf(invalidFlagError, "--%s must be length of %d", flagClusterID, key.IDLength)
		}

		matched, err := regexp.MatchString("^([a-z]+|[0-9]+)$", f.ClusterID)
		if err == nil && matched == true {
			// strings is letters only, which we also avoid
			return microerror.Maskf(invalidFlagError, "--%s must be alphanumeric", flagClusterID)
		}

		matched, err = regexp.MatchString("^[a-z0-9]+$", f.ClusterID)
		if err == nil && matched == false {
			return microerror.Maskf(invalidFlagError, "--%s must only contain [a-z0-9]", flagClusterID)
		}

		return nil
	}
	if f.Domain == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDomain)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.PodsCIDR != "" {
		if !validateCIDR(f.PodsCIDR) {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagPodsCIDR)
		}
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

	// AZ name(s)
	if len(f.MasterAZ) != 1 && len(f.MasterAZ) != 3 {
		return microerror.Maskf(invalidFlagError, "--%s must be set to either one or three availabiliy zone names", flagMasterAZ)
	}
	if !unique.StringsAreUnique(f.MasterAZ) {
		return microerror.Maskf(invalidFlagError, "--%s values must contain each AZ name only once", flagMasterAZ)
	}

	// TODO: validate that len(f.MasterAZ) == 3 is occurring in releases >= v11.5.0

	for _, az := range f.MasterAZ {
		if !aws.ValidateAZ(f.Region, az) {
			return microerror.Maskf(invalidFlagError, "The AZ name %q passed via --%s is not a valid AZ name for region %s", az, flagMasterAZ, f.Region)
		}
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

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	return true
}
