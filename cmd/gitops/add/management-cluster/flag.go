package mcluster

import (
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagDryRun          = "dry-run"
	flagInterval        = "refresh-interval"
	flagName            = "name"
	flagLocalPath       = "local-path"
	flagRepositoryeName = "repository-name"
	flagServiceAccount  = "service-account"
	flagTimeout         = "refresh-timeout"
)

type flag struct {
	DryRun         bool
	Interval       time.Duration
	Name           string
	LocalPath      string
	RepositoryName string
	ServiceAccount string
	Timeout        time.Duration
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.DryRun, flagDryRun, false, "Print files and directories instead of creating them")
	cmd.Flags().DurationVar(&f.Interval, flagInterval, time.Minute, "Source synchronization interval")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Codename of the Management Cluster")
	cmd.Flags().StringVar(&f.LocalPath, flagLocalPath, ".", "Path to the cloned GitOps repository")
	cmd.Flags().StringVar(&f.RepositoryName, flagRepositoryeName, "", "Name of the GitOps repository")
	cmd.Flags().StringVar(&f.ServiceAccount, flagServiceAccount, "automation", "Service Account for Flux to impersonate")
	cmd.Flags().DurationVar(&f.Timeout, flagTimeout, 2*time.Minute, "Synchronization timeout")
}

func (f *flag) Validate() error {
	if f.Name == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagName)
	}

	if f.RepositoryName == "" {
		return microerror.Maskf(invalidFlagsError, "--%s must not be empty", flagRepositoryeName)
	}

	return nil
}
