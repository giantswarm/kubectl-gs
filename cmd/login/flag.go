package login

import (
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagClusterAdmin   = "cluster-admin"
	flagInternalAPI    = "internal-api"
	callbackServerPort = "callback-port"

	flagWCName              = "workload-cluster"
	flagWCOrganization      = "organization"
	flagWCCertGroups        = "certificate-group"
	flagWCCertTTL           = "certificate-ttl"
	flagSelfContained       = "self-contained"
	flagWCInsecureNamespace = "insecure-namespace"
)

type flag struct {
	CallbackServerPort int
	ClusterAdmin       bool
	InternalAPI        bool

	WCName              string
	WCOrganization      string
	WCCertGroups        []string
	WCCertTTL           string
	SelfContained       string
	WCInsecureNamespace bool

	config genericclioptions.RESTClientGetter
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.CallbackServerPort, callbackServerPort, 0, "TCP port to use by the OIDC callback server. If not specified, a free port will be selected randomly.")
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Login with cluster-admin access.")
	cmd.Flags().BoolVar(&f.InternalAPI, flagInternalAPI, false, "Use Internal API in the kube config.")
	cmd.Flags().StringVar(&f.SelfContained, flagSelfContained, "", "Create a self-contained kubectl config with embedded credentials and write it to this path.")

	cmd.Flags().StringVar(&f.WCName, flagWCName, "", "Specify the name of a workload cluster to work with. If omitted, a management cluster will be accessed.")
	cmd.Flags().StringVar(&f.WCOrganization, flagWCOrganization, "", fmt.Sprintf("Organization that owns the workload cluster. Requires --%s.", flagWCName))
	cmd.Flags().StringSliceVar(&f.WCCertGroups, flagWCCertGroups, nil, fmt.Sprintf("RBAC group name to be encoded into the X.509 field \"O\". Requires --%s.", flagWCName))
	cmd.Flags().StringVar(&f.WCCertTTL, flagWCCertTTL, "1h", fmt.Sprintf(`How long the client certificate should live for. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Requires --%s.`, flagWCName))
	cmd.Flags().BoolVar(&f.WCInsecureNamespace, flagWCInsecureNamespace, false, fmt.Sprintf(`Allow using an insecure namespace for creating the client certificate. Requires --%s.`, flagWCName))

	f.config = genericclioptions.NewConfigFlags(true)
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.Flags())

	_ = cmd.Flags().MarkHidden(flagWCInsecureNamespace)
	_ = cmd.Flags().MarkHidden("namespace")
}

func (f *flag) Validate() error {
	// Validate ttl flag
	ttlFlag, err := time.ParseDuration(f.WCCertTTL)
	if err != nil {
		return microerror.Maskf(invalidFlagError, `--%s is not a valid duration. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`, flagWCCertTTL)
	}
	if ttlFlag <= 0 {
		return microerror.Maskf(invalidFlagError, `--%s cannot be negative or zero.`, flagWCCertTTL)
	}

	return nil
}
