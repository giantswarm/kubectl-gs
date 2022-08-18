package orgs

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type flag struct {
	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	return nil
}
