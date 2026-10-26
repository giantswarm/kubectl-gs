package mcluster

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagGenerateMasterKey = "gen-master-key"
	flagName              = "name"
	flagRepositoryName    = "repository-name"
	flagSkipEncryption    = "skip-encryption"
)

type flag struct {
	GenerateMasterKey bool
	Name              string
	RepositoryName    string
	SkipEncryption    bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.GenerateMasterKey, flagGenerateMasterKey, false, "Generate Management Cluster master GPG key for SOPS.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster.")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the gitops repository.")
	cmd.Flags().BoolVar(&f.SkipEncryption, flagSkipEncryption, false, "Skip the `secrets` and `.sops.keys` directories, for secrets that come from outside the repository, for example from External Secrets Operator.")
}

func (f *flag) Validate() error {
	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	if f.RepositoryName == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagRepositoryName)
	}

	if f.GenerateMasterKey && f.SkipEncryption {
		return microerror.Maskf(invalidFlagsError, "--%s and --%s are mutually exclusive", flagGenerateMasterKey, flagSkipEncryption)
	}

	return nil
}
