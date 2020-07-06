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
	"github.com/giantswarm/micrologger"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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

var (
	authScopes = [...]string{gooidc.ScopeOpenID, "profile", "email", "groups", "offline_access", "audience:server:client_id:dex-k8s-authenticator"}
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	k8sConfigAccess clientcmd.ConfigAccess

	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return microerror.Mask(invalidFlagError)
	}
	providedUrl := args[0]

	i, err := installation.New(providedUrl)
	if installation.IsUnknownUrlType(err) {
		return microerror.Maskf(unknownUrlError, "'%s' is not a valid Giant Swarm Control Plane API URL. Please check the spelling.\nIf not sure, pass the web UI URL of the installation or the installation handle as an argument instead.", providedUrl)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if installation.GetUrlType(providedUrl) == installation.UrlTypeHappa {
		fmt.Fprintf(r.stdout, color.YellowString("Note: deriving Control Plane API URl from web UI URL: %s\n", i.K8sApiURL))
	}

	oidcConfig := oidc.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Issuer:       i.AuthURL,
		RedirectURL:  fmt.Sprintf("%s:%d%s", authCallbackURL, authCallbackPort, authCallbackPath),
		AuthScopes:   authScopes[:],
	}
	auther, err := oidc.New(ctx, oidcConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	authURL := auther.GetAuthURL()
	// Open the authorization url in the user's browser, which will eventually
	// redirect the user to the local web server we'll create next.
	err = open.Run(authURL)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintf(r.stdout, "\n%s\n", color.YellowString("Your browser should now be opening this URL:"))
	fmt.Fprintf(r.stdout, "%s\n\n", authURL)

	p, err := callbackserver.RunCallbackServer(authCallbackPort, authCallbackPath, handleAuthCallback(ctx, auther))
	if err != nil {
		return microerror.Mask(err)
	}

	var authResult oidc.UserInfo
	{
		var ok bool
		authResult, ok = p.(oidc.UserInfo)
		if !ok {
			return microerror.Mask(invalidAuthResult)
		}
		authResult.ClientID = clientID
		authResult.ClientSecret = clientSecret
	}

	// Store kubeconfig.
	err = storeCredentials(r.k8sConfigAccess, i, authResult, r.fs)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintf(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.Email, i.Codename))

	contextName := generateContextName(i, authResult)
	fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected.", contextName)
	fmt.Fprintf(r.stdout, " ")
	fmt.Fprintf(r.stdout, "To switch back to this context later, use either of these commands:\n\n")
	fmt.Fprintf(r.stdout, "  kgs login %s\n", i.Codename)
	fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)

	return nil
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
	contextName := generateContextName(i, authResult)

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

func generateContextName(i *installation.Installation, authResult oidc.UserInfo) string {
	return fmt.Sprintf("%s-%s", authResult.Username, i.Codename)
}
