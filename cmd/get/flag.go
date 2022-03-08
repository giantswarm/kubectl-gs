package get

import "github.com/spf13/cobra"

const (
	flagMaxColWidth = "max-col-width"
)

type flag struct {
	MaxColWidth uint
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.PersistentFlags().UintVar(&f.MaxColWidth, flagMaxColWidth, 80, "maximum column width for output table")
}

func (f *flag) Validate() error {
	return nil
}
