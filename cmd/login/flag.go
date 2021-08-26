package login

import (
	"github.com/spf13/cobra"
)

const (
	flagClusterAdmin   = "cluster-admin"
	flagInternalAPI    = "internal-api"
	callbackServerPort = "callback-port"
)

type flag struct {
	CallbackServerPort int
	ClusterAdmin       bool
	InternalAPI        bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.CallbackServerPort, callbackServerPort, 0, "TCP port to use by the OIDC callback server. If not specified, a free port will be selected randomly.")
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Login with cluster-admin access.")
	cmd.Flags().BoolVar(&f.InternalAPI, flagInternalAPI, false, "Use Internal API in the kube config.")
}

func (f *flag) Validate() error {
	return nil
}
