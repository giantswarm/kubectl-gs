package capi

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type flag struct {
	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	f.config = genericclioptions.NewConfigFlags(true)
	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.Flags())
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	return nil
}
