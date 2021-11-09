package selfupdate

import (
	"github.com/spf13/cobra"
)

const (
	flagDryRun = "dry-run"
)

type flag struct {
	DryRun bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.DryRun, flagDryRun, false, "Only check for updates.")
}

func (f *flag) Validate() error {
	return nil
}
