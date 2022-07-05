package mcluster

import (
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagInterval       = "refresh-interval"
	flagName           = "name"
	flagRepositoryName = "repository-name"
	flagServiceAccount = "service-account"
	flagTimeout        = "refresh-timeout"
)

type flag struct {
	Interval       time.Duration
	Name           string
	RepositoryName string
	ServiceAccount string
	Timeout        time.Duration
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().DurationVar(&f.Interval, flagInterval, time.Minute, "Source synchronization interval")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryName, "", "Name of the GitOps repository")
	cmd.Flags().StringVar(&f.ServiceAccount, flagServiceAccount, "automation", "Service Account for Flux to impersonate")
	cmd.Flags().DurationVar(&f.Timeout, flagTimeout, 2*time.Minute, "Synchronization timeout")
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
