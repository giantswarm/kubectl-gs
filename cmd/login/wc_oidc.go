package login

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/v6/pkg/callbackserver"
	"github.com/giantswarm/kubectl-gs/v6/pkg/credentialcache"
	"github.com/giantswarm/kubectl-gs/v6/pkg/kubeconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/oidc"
)

var (
	directOIDCScopes = []string{"openid", "profile", "email", "offline_access"}
)

type oidcWCConfig struct {
	clusterName          string
	certCA               []byte
	controlPlaneEndpoint string
	filePath             string
	loginOptions         LoginOptions
	issuerURL            string
	oidcClientID         string
	idToken              string
	refreshToken         string
}

// createOIDCKubeconfig handles the creation of a workload cluster kubeconfig
// using direct OIDC authentication (structured authentication).
func (r *runner) createOIDCKubeconfig(ctx context.Context, k8sClient k8sclient.Interface, cluster *unstructured.Unstructured, namespace string, authConfig *structuredAuthIssuer) (string, bool, error) {
	// Fetch CA - either from flag or from MC ConfigMap.
	caData, err := fetchClusterCA(ctx, k8sClient, cluster.GetName(), namespace, r.flag.WCOIDCCAFile)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	cpHost, cpPort, err := controlPlaneEndpoint(cluster)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	clusterServer := fmt.Sprintf("https://%s:%d", cpHost, cpPort)

	// Run OIDC authentication flow.
	var authResult authInfo
	if r.flag.DeviceAuth {
		authResult, err = handleDirectDeviceOIDC(r.stdout, r.stderr, authConfig.IssuerURL, authConfig.ClientID)
	} else {
		authResult, err = handleDirectOIDC(ctx, r.stdout, r.stderr, authConfig.IssuerURL, authConfig.ClientID, r.flag.CallbackServerHost, r.flag.CallbackServerPort, r.flag.LoginTimeout)
	}
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	// Verify token against the WC API server.
	verifyErr := VerifyIDTokenWithKubernetesAPI(authResult.token, clusterServer, caData)
	if verifyErr != nil {
		_, _ = fmt.Fprintf(r.stderr, "%s\n", color.YellowString("OIDC flow succeeded but token verification returned error: %s", verifyErr.Error()))
	}

	// Warn when the IdP did not return a refresh token. The kubeconfig will
	// still work until the id_token expires, but cannot be auto-renewed
	// afterwards — the user will have to re-run 'kubectl gs login'.
	if authResult.refreshToken == "" {
		_, _ = fmt.Fprintf(r.stderr, "%s\n", color.YellowString(
			"Warning: the OIDC provider (%s) did not return a refresh token. "+
				"The generated kubeconfig will only work until the ID token expires; "+
				"after that, re-run 'kubectl gs login' to renew. "+
				"To enable automatic renewal, configure the OIDC application (client: %s) "+
				"to issue refresh tokens (enable the 'offline_access' scope and the refresh-token grant).",
			authConfig.IssuerURL, authConfig.ClientID))
	}

	wcConfig := oidcWCConfig{
		clusterName:          cluster.GetName(),
		certCA:               caData,
		controlPlaneEndpoint: clusterServer,
		filePath:             r.flag.SelfContained,
		loginOptions:         r.loginOptions,
		issuerURL:            authConfig.IssuerURL,
		oidcClientID:         authConfig.ClientID,
		idToken:              authResult.token,
		refreshToken:         authResult.refreshToken,
	}

	contextName, contextExists, err := r.storeWCOIDCCredentials(wcConfig)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	// Write tokens to credential cache for later renewal.
	if err := credentialcache.WriteWithLock(authConfig.IssuerURL, authConfig.ClientID, authResult.token, authResult.refreshToken); err != nil {
		_, _ = fmt.Fprintf(r.stderr, color.YellowString("Warning: failed to write token cache: %v\n"), err)
	}

	_, _ = fmt.Fprint(r.stdout, color.GreenString("\nCreated OIDC kubeconfig for workload cluster '%s'.\n", cluster.GetName()))

	return contextName, contextExists, nil
}

func (r *runner) storeWCOIDCCredentials(c oidcWCConfig) (string, bool, error) {
	k8sConfigAccess := r.commonConfig.GetConfigAccess()
	if r.loginOptions.selfContainedWC && c.filePath != "" {
		return printWCOIDCCredentials(k8sConfigAccess, r.fs, c, r.loginOptions.contextOverride)
	}
	return storeWCOIDCKubeconfig(k8sConfigAccess, c, r.loginOptions.contextOverride)
}

// storeWCOIDCKubeconfig saves OIDC-based WC credentials into the main kubeconfig.
func storeWCOIDCKubeconfig(k8sConfigAccess clientcmd.ConfigAccess, c oidcWCConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCOIDCKubeContextName(mcContextName, c.clusterName)
	userName := fmt.Sprintf("%s-user", contextName)
	clusterName := contextName

	contextExists := false

	{
		user, exists := config.AuthInfos[userName]
		if !exists {
			user = clientcmdapi.NewAuthInfo()
		}
		user.Exec = oidcExec(c.issuerURL, c.oidcClientID, c.idToken, c.refreshToken)
		config.AuthInfos[userName] = user
	}

	{
		cl, exists := config.Clusters[clusterName]
		if !exists {
			cl = clientcmdapi.NewCluster()
		}
		cl.Server = c.controlPlaneEndpoint
		cl.CertificateAuthority = ""
		cl.CertificateAuthorityData = c.certCA
		config.Clusters[clusterName] = cl
	}

	{
		var ctx *clientcmdapi.Context
		ctx, contextExists = config.Contexts[contextName]
		if !contextExists {
			ctx = clientcmdapi.NewContext()
		}
		ctx.Cluster = clusterName
		ctx.AuthInfo = userName
		config.Contexts[contextName] = ctx

		if c.loginOptions.switchToWCContext {
			config.CurrentContext = contextName
		} else if c.loginOptions.originContext != "" {
			config.CurrentContext = c.loginOptions.originContext
		}
	}

	err = clientcmd.ModifyConfig(k8sConfigAccess, *config, false)
	if err != nil {
		return "", contextExists, microerror.Mask(err)
	}

	return contextName, contextExists, nil
}

// printWCOIDCCredentials writes OIDC-based WC credentials into a self-contained kubeconfig file.
func printWCOIDCCredentials(k8sConfigAccess clientcmd.ConfigAccess, fs afero.Fs, c oidcWCConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCOIDCKubeContextName(mcContextName, c.clusterName)

	kc := clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			contextName: {
				Server:                   c.controlPlaneEndpoint,
				CertificateAuthorityData: c.certCA,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  contextName,
				AuthInfo: fmt.Sprintf("%s-user", contextName),
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			fmt.Sprintf("%s-user", contextName): {
				Exec: oidcExec(c.issuerURL, c.oidcClientID, c.idToken, c.refreshToken),
			},
		},
		CurrentContext: contextName,
	}

	err = mergeKubeconfigs(fs, c.filePath, kc, contextName)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	// Restore origin context if needed.
	if c.loginOptions.originContext != "" && config.CurrentContext != "" && c.loginOptions.originContext != config.CurrentContext {
		config.CurrentContext = c.loginOptions.originContext
		err = clientcmd.ModifyConfig(k8sConfigAccess, *config, false)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
	}

	return contextName, false, nil
}

// handleDirectOIDC performs the browser-based OIDC flow directly against
// a non-Dex issuer (e.g., Azure AD via structured authentication).
func handleDirectOIDC(ctx context.Context, out io.Writer, errOut io.Writer, issuerURL, oidcClientID, host string, port int, timeout time.Duration) (authInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var err error
	var authProxy *callbackserver.CallbackServer
	{
		// For direct OIDC (non-Dex) providers like Azure AD, the redirect URI
		// registered in the provider is typically just http://localhost:<port>
		// without a path suffix. Use "/" as the callback path to match.
		config := callbackserver.Config{
			Host:              host,
			Port:              port,
			RedirectURI:       "/",
			ReadHeaderTimeout: oidcReadHeaderTimeout,
		}
		authProxy, err = callbackserver.New(config)
		if err != nil {
			return authInfo{}, microerror.Mask(err)
		}
	}

	oidcConfig := oidc.Config{
		ClientID:    oidcClientID,
		Issuer:      issuerURL,
		RedirectURL: fmt.Sprintf("%s:%d", oidcCallbackURL, authProxy.Port()),
		AuthScopes:  directOIDCScopes,
	}
	auther, err := oidc.New(ctx, oidcConfig)
	if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	authURL := auther.GetPlainAuthURL()

	_, _ = fmt.Fprintf(out, "\n%s\n", color.YellowString("Your browser should now be opening this URL:"))
	_, _ = fmt.Fprintf(out, "%s\n\n", authURL)

	err = open.Start(authURL)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "%s\n\n", color.YellowString("Couldn't open the default browser. Please access the URL above to continue logging in."))
	}

	p, err := authProxy.Run(ctx, handleOIDCCallback(ctx, auther))
	if callbackserver.IsTimedOut(err) {
		return authInfo{}, microerror.Maskf(authResponseTimedOutError, "failed to get an authentication response on time")
	} else if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	user, ok := p.(oidc.UserInfo)
	if !ok {
		return authInfo{}, microerror.Mask(invalidAuthResult)
	}

	return authInfo{
		username:     user.Username,
		token:        user.IDToken,
		refreshToken: user.RefreshToken,
		clientID:     oidcClientID,
	}, nil
}

// handleDirectDeviceOIDC performs the device auth flow directly against a non-Dex issuer.
func handleDirectDeviceOIDC(out io.Writer, errOut io.Writer, issuerURL, oidcClientID string) (authInfo, error) {
	_, _ = fmt.Fprintf(errOut, "%s\n", color.YellowString("Device authentication flow is not yet supported for direct OIDC issuers. Please use browser-based login instead."))
	return authInfo{}, microerror.Maskf(deviceAuthError, "device auth is not supported for direct OIDC issuers; use browser-based login")
}
