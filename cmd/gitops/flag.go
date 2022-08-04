package gitops

import "github.com/spf13/cobra"

const (
	flagDryRun    = "dry-run"
	flagLocalPath = "local-path"
)

type flag struct {
	DryRun    bool
	LocalPath string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&f.DryRun, flagDryRun, false, "Print files and directories instead of creating them")
	cmd.PersistentFlags().StringVar(&f.LocalPath, flagLocalPath, ".", "Path to the cloned GitOps repository")
}

func (f *flag) Validate() error {
	return nil
}
