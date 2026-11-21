package org

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagName              = "name"
	flagManagementCluster = "management-cluster"
	flagRepositoryName    = "repository-name"
)

type flag struct {
	ManagementCluster string
	Name              string
	RepositoryName    string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Management Cluster the Organization belongs to.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Organization name.")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the GitRepository the organization's namespace Kustomization reads. Defaults to the one the management cluster's Kustomization reads.")
}

func (f *flag) Validate() error {
	missing := firstEmpty([]requiredFlag{
		{name: flagManagementCluster, value: f.ManagementCluster},
		{name: flagName, value: f.Name},
	})
	if missing != "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", missing)
	}

	return nil
}

type requiredFlag struct {
	name  string
	value string
}

// firstEmpty returns the name of the first flag without a value, or an empty
// string when they all have one.
func firstEmpty(flags []requiredFlag) string {
	for _, f := range flags {
		if f.value == "" {
			return f.name
		}
	}

	return ""
}
