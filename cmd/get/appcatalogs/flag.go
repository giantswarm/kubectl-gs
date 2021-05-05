package appcatalogs

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagLabelSelector     = "selector"
	publicCatalogSelector = "application.giantswarm.io/catalog-visibility=public"
)

type flag struct {
	LabelSelector string

	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.LabelSelector, flagLabelSelector, "l", publicCatalogSelector, "Specify label selector(s) to filter Apps by.")

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
