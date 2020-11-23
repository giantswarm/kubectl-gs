package login

import (
	"github.com/spf13/cobra"
)

const (
	flagClusterAdmin = "cluster-admin"
)

type flag struct {
	ClusterAdmin bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Login with cluster-admin access.")
}

func (f *flag) Validate() error {
	return nil
}
