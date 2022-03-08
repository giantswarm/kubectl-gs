package catalogs

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagAllNamespaces = "all-namespaces"
	flagMaxColWidth   = "max-col-width"
)

type flag struct {
	AllNamespaces bool
	MaxColWidth   uint

	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&f.AllNamespaces, flagAllNamespaces, "A", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	cmd.Flags().UintVar(&f.MaxColWidth, flagMaxColWidth, 80, "maximum column width for output table")

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
