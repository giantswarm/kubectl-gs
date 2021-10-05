package login

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagClusterAdmin   = "cluster-admin"
	flagInternalAPI    = "internal-api"
	callbackServerPort = "callback-port"

	flagKeypairCluster      = "keypair-cluster"
	flagKeypairOrganization = "keypair-organization"
	flagKeypairGroups       = "keypair-group"
	flagKeypairTTL          = "keypair-ttl"
)

type flag struct {
	CallbackServerPort int
	ClusterAdmin       bool
	InternalAPI        bool

	KeypairCluster      string
	KeypairOrganization string
	KeypairGroups       []string
	KeypairTTL          string

	config genericclioptions.RESTClientGetter
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.CallbackServerPort, callbackServerPort, 0, "TCP port to use by the OIDC callback server. If not specified, a free port will be selected randomly.")
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Login with cluster-admin access.")
	cmd.Flags().BoolVar(&f.InternalAPI, flagInternalAPI, false, "Use Internal API in the kube config.")

	cmd.Flags().StringVar(&f.KeypairCluster, flagKeypairCluster, "", "The name of the workload cluster to create a keypair for.")
	cmd.Flags().StringVar(&f.KeypairOrganization, flagKeypairOrganization, "", "The organization that the cluster belongs to.")
	cmd.Flags().StringSliceVar(&f.KeypairGroups, flagKeypairGroups, nil, "The group names (O) to set in the X.509 certificate.")
	cmd.Flags().StringVar(&f.KeypairTTL, flagKeypairTTL, "1h", "How long the keypair should live for.")

	f.config = genericclioptions.NewConfigFlags(true)
}

func (f *flag) Validate() error {
	if len(f.KeypairCluster) > 0 && len(f.KeypairOrganization) < 1 {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty when --%s is provided.", flagKeypairOrganization, flagKeypairCluster)
	}

	return nil
}
