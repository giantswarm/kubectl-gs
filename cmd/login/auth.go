package login

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
)

func handleAuth(ctx context.Context, out io.Writer, i *installation.Installation) (oidc.UserInfo, error) {
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

	authURL := auther.GetAuthURL()
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

func storeCredentials(k8sConfigAccess clientcmd.ConfigAccess, i *installation.Installation, authResult oidc.UserInfo, fs afero.Fs) error {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	err = kubeconfig.WriteCertificate(i.CACert, i.Codename, fs)
	if err != nil {
		return microerror.Mask(err)
	}

	kUsername := fmt.Sprintf("%s-%s", authResult.Username, i.Codename)
	contextName := generateKubeContextName(i.Codename)

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
		initialCluster, exists := config.Clusters[i.Codename]
		if !exists {
			initialCluster = clientcmdapi.NewCluster()
		}

		initialCluster.Server = i.K8sApiURL

		var certPath string
		certPath, err = kubeconfig.GetKubeCertFilePath(i.Codename)
		if err != nil {
			return microerror.Mask(err)
		}
		initialCluster.CertificateAuthority = certPath

		// Add cluster configuration to config.
		config.Clusters[i.Codename] = initialCluster
	}

	{
		// Create authenticated context.
		initialContext, exists := config.Contexts[kUsername]
		if !exists {
			initialContext = clientcmdapi.NewContext()
		}

		initialContext.Cluster = i.Codename

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
