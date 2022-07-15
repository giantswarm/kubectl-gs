package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagDebug               = "debug"
	flagDisableVersionCheck = "disable-version-check"
)

type flag struct {
	config              genericclioptions.RESTClientGetter
	disableVersionCheck bool
}

func (f *flag) Init(cmd *cobra.Command) {
	// This value is ignored. The real value is handled inside 'main.go'.
	cmd.PersistentFlags().Bool(flagDebug, false, "Toggle debug mode, for seeing full error output.")
	cmd.PersistentFlags().BoolVar(&f.disableVersionCheck, flagDisableVersionCheck, false, "Disable self-update version check.")

	f.config = genericclioptions.NewConfigFlags(true)
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.PersistentFlags())
}

func (f *flag) Validate() error {
	return nil
}
