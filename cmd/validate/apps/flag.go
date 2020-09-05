package apps

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagAllNamespaces    = "all-namespaces"
	flagLabelSelector    = "selector"
	flagQuiet            = "quiet"
	flagValuesSchemaFile = "values-schema-file"
)

type flag struct {
	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags

	AllNamespaces    bool
	LabelSelector    string
	Quiet            bool
	ValuesSchemaFile string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&f.Quiet, flagQuiet, "q", false, "Suppress output and just return the exit code.")
	cmd.Flags().StringVarP(&f.LabelSelector, flagLabelSelector, "l", "", "Specify label selector(s) to filter Apps by.")
	cmd.Flags().BoolVarP(&f.AllNamespaces, flagAllNamespaces, "A", false, "Validate apps across all namespaces. This can take a long time.")
	cmd.Flags().StringVarP(&f.ValuesSchemaFile, flagValuesSchemaFile, "f", "", "Provide your own schema file to validate app values against.")

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
