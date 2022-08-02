package org

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagName              = "name"
	flagManagementCluster = "management-cluster"
)

type flag struct {
	ManagementCluster string
	Name              string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Management Cluster the Organization belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Organization name.")
}

func (f *flag) Validate() error {
	if f.ManagementCluster == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagManagementCluster)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	return nil
}
