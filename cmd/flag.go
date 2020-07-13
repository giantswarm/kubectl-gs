package cmd

import (
	"github.com/spf13/cobra"
)

const (
	flagDebugMode = "debug"
)

type flag struct {
	debugMode *bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(f.debugMode, flagDebugMode, false, "Toggle debug mode, for seeing full error output.")
}

func (f *flag) Validate() error {
	return nil
}
