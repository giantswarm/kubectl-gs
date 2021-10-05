package organization

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagName   = "name"
	flagOutput = "output"
)

type flag struct {
	Name   string
	Output string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Name, flagName, "", "Organization name.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for the resulting manifest. (default: stdout)")
}

func (f *flag) Validate() error {
	if len(f.Name) < 1 {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}

	return nil
}
