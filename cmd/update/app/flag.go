package app

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagVersion = "version"
	flagName    = "name"
)

type flag struct {
	config  genericclioptions.RESTClientGetter
	print   *genericclioptions.PrintFlags
	Name    string
	Version string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "Version to update the app to")
	// Hide flag in favour of the longDescription, otherwise if the number of supported
	// update flags grows, it may be hard to differentiate them from the rest of the flags,
	// like kubectl global flags.
	_ = cmd.Flags().MarkHidden(flagVersion)

	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the app to update")
	_ = cmd.Flags().MarkHidden(flagName)

	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}

	if f.Version == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagVersion)
	}

	return nil
}
