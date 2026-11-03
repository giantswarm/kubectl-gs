package upgrade

import (
	"github.com/spf13/cobra"
)

const (
	flagForce = "force"
)

// flag holds the flags of `upgrade`. It also uses the --local-path and
// --dry-run flags it inherits from `gitops`.
type flag struct {
	Force bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.Force, flagForce, false, "Upgrade even when the repository has uncommitted changes, or is not a git repository, which leaves no safe way to undo a failed upgrade")
}

func (f *flag) Validate() error {
	return nil
}
