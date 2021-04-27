package login

import (
	"github.com/spf13/cobra"
)

const (
	flagClusterAdmin = "cluster-admin"
	flagInternalAPI  = "internal-api"
)

type flag struct {
	ClusterAdmin bool
	InternalAPI  bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Login with cluster-admin access.")
	cmd.Flags().BoolVar(&f.InternalAPI, flagInternalAPI, false, "Use Internal API in the kube config.")
}

func (f *flag) Validate() error {
	return nil
}
