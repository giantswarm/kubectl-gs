package releases

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type flag struct {
	ActiveOnly bool

	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.ActiveOnly, "active-only", false, "Only show active releases")

	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	return nil
}
