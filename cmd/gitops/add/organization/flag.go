package mcluster

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagName   = "name"
	flagMCName = "management-cluster"
)

type flag struct {
	MCName string
	Name   string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.MCName, flagMCName, "", "Management Cluster the Organization belongs to")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Organization name")
}

func (f *flag) Validate() error {
	if f.MCName == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagMCName)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	return nil
}
