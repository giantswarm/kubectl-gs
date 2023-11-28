package policyexception

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagDraft  = "draft"
	flagOutput = "output"
)

type flag struct {
	Draft  string
	Output string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Draft, flagDraft, "", "PolicyExceptionDraft name.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for the resulting manifest. (default: stdout)")
}

func (f *flag) Validate() error {
	if len(f.Draft) < 1 {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagDraft)
	}

	return nil
}
