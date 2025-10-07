package login

import (
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagClusterAdmin       = "cluster-admin"
	flagInternalAPI        = "internal-api"
	flagCallbackServerHost = "callback-host"
	flagCallbackServerPort = "callback-port"
	flagKeepContext        = "keep-context"

	flagWCName              = "workload-cluster"
	flagWCOrganization      = "organization"
	flagWCCertCNPrefix      = "cn-prefix"
	flagWCCertGroups        = "certificate-group"
	flagWCCertTTL           = "certificate-ttl"
	flagSelfContained       = "self-contained"
	flagWCInsecureNamespace = "insecure-namespace"

	flagConnectorID = "connector-id"

	flagProxy     = "proxy"
	flagProxyPort = "proxy-port"

	flagAwsProfile = "aws-profile"

	flagLoginTimeout = "login-timeout"

	flagDeviceAuth = "device-auth"

	envKeepContext = "KUBECTL_GS_LOGIN_KEEP_CONTEXT"
)

type flag struct {
	CallbackServerHost string
	CallbackServerPort int
	ClusterAdmin       bool
	InternalAPI        bool
	KeepContext        bool

	WCName              string
	WCOrganization      string
	WCCertCNPrefix      string
	WCCertGroups        []string
	WCCertTTL           string
	SelfContained       string
	WCInsecureNamespace bool

	ConnectorID string

	Proxy     bool
	ProxyPort int

	AWSProfile string

	LoginTimeout time.Duration

	DeviceAuth bool
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.CallbackServerHost, flagCallbackServerHost, "localhost", "Address to listen on for the OIDC callback server. If not specified, it only listens on 'localhost'. Use an empty value or '0.0.0.0' to listen on all interfaces. The redirect URL will still contain 'http://localhost' since that is the allowed URL.")
	cmd.Flags().IntVar(&f.CallbackServerPort, flagCallbackServerPort, 0, "TCP port to use by the OIDC callback server. If not specified, a free port will be selected randomly.")
	cmd.Flags().BoolVar(&f.ClusterAdmin, flagClusterAdmin, false, "Log in as Giant Swarm staff member.")
	cmd.Flags().BoolVar(&f.InternalAPI, flagInternalAPI, false, "Use Internal API in the kube config. Please check the documentation for more details.")
	cmd.Flags().StringVar(&f.SelfContained, flagSelfContained, "", "Create a self-contained kubectl config with embedded credentials and write it to this path.")

	viper.AutomaticEnv()
	keepContextDefault := viper.GetBool(envKeepContext)
	cmd.Flags().BoolVar(&f.KeepContext, flagKeepContext, keepContextDefault, "Keep the current kubectl context. If not set/false, will set the current-context to the one representing the cluster to log in to.")

	cmd.Flags().StringVar(&f.WCName, flagWCName, "", "For client certificate creation. Specify the name of a workload cluster to work with. If omitted, a management cluster will be accessed.")
	cmd.Flags().StringVar(&f.WCOrganization, flagWCOrganization, "", fmt.Sprintf("For client certificate creation. Organization that owns the workload cluster. Requires --%s.", flagWCName))
	cmd.Flags().StringSliceVar(&f.WCCertGroups, flagWCCertGroups, nil, fmt.Sprintf("For client certificate creation. RBAC group name to be encoded into the X.509 field \"O\". Requires --%s.", flagWCName))
	cmd.Flags().StringVar(&f.WCCertTTL, flagWCCertTTL, "1h", fmt.Sprintf(`For client certificate creation. How long the client certificate should live for. Valid time units are "ms", "s", "m", "h". Requires --%s.`, flagWCName))
	cmd.Flags().StringVar(&f.WCCertCNPrefix, flagWCCertCNPrefix, "", fmt.Sprintf(`For client certificate creation. Prefix for the name encoded in the X.509 field "CN". Requires --%s.`, flagWCName))
	cmd.Flags().BoolVar(&f.WCInsecureNamespace, flagWCInsecureNamespace, false, fmt.Sprintf(`For client certificate creation. Allow using an insecure namespace for creating the client certificate. Requires --%s.`, flagWCName))

	cmd.Flags().StringVar(&f.ConnectorID, flagConnectorID, "", "Dex connector to use for authentication. This allows to skip the selection page.")

	cmd.Flags().BoolVar(&f.Proxy, flagProxy, false, "Enable socks proxy configuration for the cluster. Only Supported for Workload Cluster using clientcert auth mode")
	cmd.Flags().IntVar(&f.ProxyPort, flagProxyPort, 9000, "Port for the socks proxy configuration for the cluster")

	cmd.Flags().StringVar(&f.AWSProfile, flagAwsProfile, "", "AWS profile name that the created kubeconfig will always use, Only applicable for EKS clusters.")

	cmd.Flags().DurationVar(&f.LoginTimeout, flagLoginTimeout, 60*time.Second, "Duration for which kubectl gs will wait for the OIDC login to complete. Once the timeout is reached, OIDC login will fail.")

	cmd.Flags().BoolVar(&f.DeviceAuth, flagDeviceAuth, false, "Use device authentication flow to log in")

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

	if f.LoginTimeout <= 0 {
		return microerror.Maskf(invalidFlagError, `--%s cannot be negative or zero`, flagLoginTimeout)
	}

	keepContextEnvVar := viper.GetString(envKeepContext)
	if keepContextEnvVar != "" && keepContextEnvVar != "true" && keepContextEnvVar != "false" {
		return microerror.Maskf(invalidFlagError, "KUBECTL_GS_LOGIN_KEEP_CONTEXT environment variable must be either 'true' or 'false'")
	}

	return nil
}
