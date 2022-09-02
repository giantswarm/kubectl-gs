package mcluster

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagGenerateMasterKey = "gen-master-key"
	flagName              = "name"
	flagRepositoryName    = "repository-name"
)

type flag struct {
	GenerateMasterKey bool
	Name              string
	RepositoryName    string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.GenerateMasterKey, flagGenerateMasterKey, false, "Generate Management Cluster master GPG key for SOPS.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster.")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the gitops repository.")
}

func (f *flag) Validate() error {
	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	if f.RepositoryName == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagRepositoryName)
	}

	return nil
}
