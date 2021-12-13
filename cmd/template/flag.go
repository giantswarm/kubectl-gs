package template

import (
	"github.com/spf13/cobra"
)

const (
	flagFromFile = "from-file"
)

type flag struct {
	FromFile string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.FromFile, flagFromFile, "", "Do from file.")
}

func (f *flag) Validate() error {

	return nil
}
