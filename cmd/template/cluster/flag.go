package cluster

import (
	"net"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/mpvl/unique"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/aws"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
	"github.com/giantswarm/kubectl-gs/pkg/release"
)

const (
	flagClusterID    = "cluster-id"
	flagCredential   = "credential"
	flagDomain       = "domain"
	flagMasterAZ     = "master-az"
	flagName         = "name"
	flagPodsCIDR     = "pods-cidr"
	flagExternalSNAT = "external-snat"
	flagOutput       = "output"
	flagOwner        = "owner"
	flagRegion       = "region"
	flagRelease      = "release"
	flagLabel        = "label"
)

type flag struct {
	ClusterID    string
	Credential   string
	Domain       string
	MasterAZ     []string
	Name         string
	PodsCIDR     string
	ExternalSNAT bool
	Output       string
	Owner        string
	Region       string
	Release      string
	Label        []string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Domain, flagDomain, "", "Installation base domain.")
	cmd.Flags().StringVar(&f.ClusterID, flagClusterID, "", "User-defined cluster ID.")
	cmd.Flags().StringVar(&f.Credential, flagCredential, "credential-default", "Cloud provider credentials used to spin up the cluster.")
	cmd.Flags().BoolVar(&f.ExternalSNAT, flagExternalSNAT, false, "AWS CNI configuration.")
	cmd.Flags().StringSliceVar(&f.MasterAZ, flagMasterAZ, []string{}, "Tenant master availability zone.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Tenant cluster name.")
	cmd.Flags().StringVar(&f.PodsCIDR, flagPodsCIDR, "", "CIDR used for the pods.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region (e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
	cmd.Flags().StringSliceVar(&f.Label, flagLabel, nil, "Tenant cluster label.")
}

func (f *flag) Validate() error {
	var err error

	if f.ClusterID != "" {
		if len(f.ClusterID) != key.IDLength {
			return microerror.Maskf(invalidFlagError, "--%s must be length of %d", flagClusterID, key.IDLength)
		}

		matched, err := regexp.MatchString("^([a-z]+|[0-9]+)$", f.ClusterID)
		if err == nil && matched {
			// strings is letters only, which we also avoid
			return microerror.Maskf(invalidFlagError, "--%s must be alphanumeric", flagClusterID)
		}

		matched, err = regexp.MatchString("^[a-z0-9]+$", f.ClusterID)
		if err == nil && !matched {
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

	_, err = clusterlabels.Parse(f.Label)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagLabel, err)
	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}
