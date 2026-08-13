package check

import (
	"github.com/spf13/cobra"
)

const (
	flagAdopt = "adopt"
)

type flag struct {
	Adopt bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.Adopt, flagAdopt, false, "Record the layers of a repository that does not track its structure version yet.")
}

func (f *flag) Validate() error {
	return nil
}
