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
	flagOwner                   = "owner"
	flagRegion                  = "region"
	flagRelease                 = "release"
	flagTemplateDefaultNodepool = "create-default-nodepool"
)

type flag struct {
	Domain                  string
	MasterAZ                string
	Name                    string
	NoCache                 bool
	Owner                   string
	Region                  string
	Release                 string
	TemplateDefaultNodepool bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Domain, flagDomain, "", "Installation base domain.")
	cmd.Flags().StringVar(&f.MasterAZ, flagMasterAZ, "", "Tenant master availability zone.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Tenant cluster name.")
	cmd.Flags().BoolVar(&f.NoCache, flagNoCache, false, "Force updating release folder.")
	cmd.Flags().StringVar(&f.Owner, flagOwner, "", "Tenant cluster owner organization.")
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "Installation region(e.g. eu-central-1).")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Tenant cluster release.")
	cmd.Flags().BoolVar(&f.TemplateDefaultNodepool, flagTemplateDefaultNodepool, true, "Template default nodepool CRs with cluster CRs.")
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
	// super basic AZ name validation
	if len(strings.Split(f.MasterAZ, "-")) != 3 || !strings.Contains(f.MasterAZ, f.Region) {
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

	return nil
}
