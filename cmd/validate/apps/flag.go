package apps

import (
	"github.com/spf13/cobra"
)

const (
	flagAllNamespaces    = "all-namespaces"
	flagLabelSelector    = "selector"
	flagQuiet            = "quiet"
	flagOutputFormat     = "output"
	flagValuesSchemaFile = "values-schema-file"
)

type flag struct {
	AllNamespaces    bool
	LabelSelector    string
	OutputFormat     string
	Quiet            bool
	ValuesSchemaFile string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&f.Quiet, flagQuiet, "q", false, "Suppress output and just return the exit code.")
	cmd.Flags().StringVarP(&f.LabelSelector, flagLabelSelector, "l", "", "Specify label selector(s) to filter Apps by.")
	cmd.Flags().BoolVarP(&f.AllNamespaces, flagAllNamespaces, "A", false, "Validate apps across all namespaces. This can take a long time.")
	cmd.Flags().StringVarP(&f.ValuesSchemaFile, flagValuesSchemaFile, "f", "", "Provide your own schema file to validate app values against.")
	cmd.Flags().StringVarP(&f.OutputFormat, flagOutputFormat, "o", "", "Output format. Use 'report' to get a human readable report of validation issues.")
}

func (f *flag) Validate() error {
	return nil
}
