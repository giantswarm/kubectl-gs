package cluster

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagName           = "name"
	flagReleaseVersion = "release-version"
	flagScheduledTime  = "scheduled-time"
	flagProvider       = "provider"
)

type flag struct {
	config         genericclioptions.RESTClientGetter
	print          *genericclioptions.PrintFlags
	Name           string
	ReleaseVersion string
	ScheduledTime  string
	Provider       string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the cluster to update.")
	_ = cmd.Flags().MarkHidden(flagName)

	cmd.Flags().StringVar(&f.ReleaseVersion, flagReleaseVersion, "", "Update the cluster to a release version. The release version must be higher than the current release version.")
	_ = cmd.Flags().MarkHidden(flagReleaseVersion)

	cmd.Flags().StringVar(&f.ScheduledTime, flagScheduledTime, "", "Optionally: Scheduled time when cluster should be updated. The value has to be in RFC822 Format and UTC time zone.")
	_ = cmd.Flags().MarkHidden(flagScheduledTime)

	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Name of the provider.")
	_ = cmd.Flags().MarkHidden(flagProvider)

	f.config = genericclioptions.NewConfigFlags(true)
	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.Flags())
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}

	if f.ReleaseVersion == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagReleaseVersion)
	}

	return nil
}
