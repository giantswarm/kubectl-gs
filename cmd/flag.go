package cmd

import (
	"github.com/spf13/cobra"
)

const (
	flagDebug = "debug"
)

type flag struct {
}

func (f *flag) Init(cmd *cobra.Command) {
	// This value is ignored. The real value is handled inside 'main.go'.
	cmd.PersistentFlags().Bool(flagDebug, false, "Toggle debug mode, for seeing full error output.")
}

func (f *flag) Validate() error {
	return nil
}
