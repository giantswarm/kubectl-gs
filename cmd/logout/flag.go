package logout

import (
	"github.com/spf13/cobra"
)

const (
	flagKeepRefreshToken = "keep-refresh-token"
)

type flag struct {
	KeepRefreshToken bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.KeepRefreshToken, flagKeepRefreshToken, false, "Set to true to keep the refresh token. Otherwise it will be removed.")
}

func (f *flag) Validate() error {
	return nil
}
