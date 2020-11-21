package login

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	gooidc "github.com/coreos/go-oidc"
	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/afero"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/cmd/login/template"
	"github.com/giantswarm/kubectl-gs/pkg/callbackserver"
	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
	"github.com/giantswarm/kubectl-gs/pkg/oidc"
)

const (
	clientID     = "zQiFLUnrTFQwrybYzeY53hWWfhOKWRAU"
	clientSecret = "TVHzVPin2WTiCma6bqp5hdxgKbWZKRbz" // #nosec G101

	authCallbackURL  = "http://localhost"
	authCallbackPort = 8085
	authCallbackPath = "/oauth/callback"

	customerConnectorID   = "customer"
	giantswarmConnectorID = "giantswarm"
)

var (
	authScopes = [...]string{gooidc.ScopeOpenID, "profile", "email", "groups", "offline_access", "audience:server:client_id:dex-k8s-authenticator"}
)

// handleAuth executes the OIDC authentication against an installation's authentication provider.
func handleAuth(ctx context.Context, out io.Writer, i *installation.Installation, clusterAdmin bool) (oidc.UserInfo, error) {
	oidcConfig := oidc.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Issuer:       i.AuthURL,
		RedirectURL:  fmt.Sprintf("%s:%d%s", authCallbackURL, authCallbackPort, authCallbackPath),
		AuthScopes:   authScopes[:],
	}
	auther, err := oidc.New(ctx, oidcConfig)
	if err != nil {
		return oidc.UserInfo{}, microerror.Mask(err)
	}

	// select dex connector_id based on clusterAdmin value
	var connectorID string
	{
		if clusterAdmin {
			connectorID = giantswarmConnectorID
		} else {
			connectorID = customerConnectorID
		}
	}
	authURL := fmt.Sprintf("%s&connector_id=%s", auther.GetAuthURL(), connectorID)
	{
		// Open the authorization url in the user's browser, which will eventually
		// redirect the user to the local web server we'll create next.
		err = open.Run(authURL)
		if err != nil {
			return oidc.UserInfo{}, microerror.Mask(err)
		}
	}

	fmt.Fprintf(out, "\n%s\n", color.YellowString("Your browser should now be opening this URL:"))
	fmt.Fprintf(out, "%s\n\n", authURL)

	// Create a local web server, for fetching all the authentication data from
	// the authentication provider.
	p, err := callbackserver.RunCallbackServer(authCallbackPort, authCallbackPath, handleAuthCallback(ctx, auther))
	if err != nil {
		return oidc.UserInfo{}, microerror.Mask(err)
	}

	var authResult oidc.UserInfo
	{
		var ok bool
		authResult, ok = p.(oidc.UserInfo)
		if !ok {
			return oidc.UserInfo{}, microerror.Mask(invalidAuthResult)
		}
		authResult.ClientID = clientID
		authResult.ClientSecret = clientSecret
	}

	return authResult, nil
}

// handleAuthCallback is the callback executed after the authentication response was
// received from the authentication provider.
func handleAuthCallback(ctx context.Context, a *oidc.Authenticator) func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
		res, err := a.HandleIssuerResponse(ctx, r.URL.Query().Get("state"), r.URL.Query().Get("code"))
		if err != nil {
			failureTemplate, tErr := template.GetFailedHTMLTemplateReader()
			if tErr != nil {
				return oidc.UserInfo{}, microerror.Mask(tErr)
			}

			w.Header().Set("Content-Type", "text/html")
			http.ServeContent(w, r, "", time.Time{}, failureTemplate)

			return oidc.UserInfo{}, microerror.Mask(err)
		}

		successTemplate, err := template.GetSuccessHTMLTemplateReader()
		if err != nil {
			return oidc.UserInfo{}, microerror.Mask(err)
		}

		w.Header().Set("Content-Type", "text/html")
		http.ServeContent(w, r, "", time.Time{}, successTemplate)

		return res, nil
	}
}

// storeCredentials stores the installation's CA certificate, and
// updates the kubeconfig with the configuration for the k8s api access.
func storeCredentials(k8sConfigAccess clientcmd.ConfigAccess, i *installation.Installation, authResult oidc.UserInfo, fs afero.Fs) error {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	kUsername := fmt.Sprintf("gs-%s-%s", authResult.Username, i.Codename)
	contextName := kubeconfig.GenerateKubeContextName(i.Codename)
	clusterName := fmt.Sprintf("gs-%s", i.Codename)

	// Store CA certificate.
	err = kubeconfig.WriteCertificate(i.CACert, clusterName, fs)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		// Create authenticated user.
		initialUser, exists := config.AuthInfos[kUsername]
		if !exists {
			initialUser = clientcmdapi.NewAuthInfo()
		}

		initialUser.AuthProvider = &clientcmdapi.AuthProviderConfig{
			Name: "oidc",
			Config: map[string]string{
				"client-id":      authResult.ClientID,
				"client-secret":  authResult.ClientSecret,
				"id-token":       authResult.IDToken,
				"idp-issuer-url": i.AuthURL,
				"refresh-token":  authResult.RefreshToken,
			},
		}

		// Add user information to config.
		config.AuthInfos[kUsername] = initialUser
	}

	{
		// Create authenticated cluster.
		initialCluster, exists := config.Clusters[clusterName]
		if !exists {
			initialCluster = clientcmdapi.NewCluster()
		}

		initialCluster.Server = i.K8sApiURL

		var certPath string
		certPath, err = kubeconfig.GetKubeCertFilePath(clusterName)
		if err != nil {
			return microerror.Mask(err)
		}
		initialCluster.CertificateAuthority = certPath

		// Add cluster configuration to config.
		config.Clusters[clusterName] = initialCluster
	}

	{
		// Create authenticated context.
		initialContext, exists := config.Contexts[contextName]
		if !exists {
			initialContext = clientcmdapi.NewContext()
		}

		initialContext.Cluster = clusterName

		initialContext.AuthInfo = kUsername

		// Add context configuration to config.
		config.Contexts[contextName] = initialContext

		// Select newly created context as current.
		config.CurrentContext = contextName
	}

	err = clientcmd.ModifyConfig(k8sConfigAccess, *config, true)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// switchContext modifies the existing kubeconfig, and switches the currently
// active context to the one specified.
func switchContext(ctx context.Context, k8sConfigAccess clientcmd.ConfigAccess, newContextName string) error {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	if newContextName == config.CurrentContext {
		return microerror.Mask(contextAlreadySelectedError)
	}

	// Check if the context exists.
	if _, exists := config.Contexts[newContextName]; !exists {
		return microerror.Maskf(contextDoesNotExistError, "There is no context named '%s'. Please make sure you spelled the installation handle correctly.\nIf not sure, pass the Control Plane API URL or the web UI URL of the installation as an argument.", newContextName)
	}

	authProvider, exists := kubeconfig.GetAuthProvider(config, newContextName)
	if !exists {
		return microerror.Maskf(incorrectConfigurationError, "There is no authentication configuration for the '%s' context", newContextName)
	}

	var auther *oidc.Authenticator
	{
		oidcConfig := oidc.Config{
			Issuer:       authProvider.Config["idp-issuer-url"],
			ClientID:     authProvider.Config["client-id"],
			ClientSecret: authProvider.Config["client-secret"],
		}
		auther, err = oidc.New(ctx, oidcConfig)
		if err != nil {
			return microerror.Maskf(incorrectConfigurationError, "\n%v", err.Error())
		}
	}

	// Renew authentication token.
	{
		idToken, rToken, err := auther.RenewToken(ctx, authProvider.Config["refresh-token"])
		if err != nil {
			return microerror.Mask(tokenRenewalFailedError)
		}
		authProvider.Config["refresh-token"] = rToken
		authProvider.Config["id-token"] = idToken
	}

	config.CurrentContext = newContextName

	err = clientcmd.ModifyConfig(k8sConfigAccess, *config, true)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func isLoggedWithGSContext(k8sConfigAccess clientcmd.ConfigAccess) (string, bool) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false
	}

	if !kubeconfig.IsKubeContext(config.CurrentContext) {
		return config.CurrentContext, false
	}

	return config.CurrentContext, true
}
