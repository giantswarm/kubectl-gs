package login

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/pkg/graphql"
	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	k8sConfigAccess clientcmd.ConfigAccess
	loginOptions    LoginOptions

	stdout io.Writer
	stderr io.Writer
}

type LoginOptions struct {
	isWCLogin         bool
	selfContainedMC   bool
	selfContainedWC   bool
	switchToMCcontext bool
	switchToWCcontext bool
	originContext     string
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	r.setLoginOptions(ctx)
	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	if len(args) < 1 {
		err = r.tryToReuseExistingContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	// This can be a kubernetes context name,
	// installation code name, or happa/k8s api URL.
	installationIdentifier := strings.ToLower(args[0])

	if _, contextType := kubeconfig.IsKubeContext(installationIdentifier); contextType == kubeconfig.ContextTypeMC {
		err = r.loginWithKubeContextName(ctx, installationIdentifier)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if kubeconfig.IsCodeName(installationIdentifier) {
		err = r.loginWithCodeName(ctx, installationIdentifier)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		var tokenOverride string
		if c, ok := r.flag.config.(*genericclioptions.ConfigFlags); ok && c.BearerToken != nil && len(*c.BearerToken) > 0 {
			tokenOverride = *c.BearerToken
		}

		err = r.loginWithURL(ctx, installationIdentifier, true, tokenOverride)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if r.loginOptions.isWCLogin {
		err = r.createClusterClientCert(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
func (r *runner) tryToGetCurrentContext(ctx context.Context) (string, error) {
	config, err := r.k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}
	return config.CurrentContext, nil
}

func (r *runner) setLoginOptions(ctx context.Context) {
	originContext, err := r.tryToGetCurrentContext(ctx)
	if err != nil {
		fmt.Fprintln(r.stdout, color.YellowString("Failed trying to determine current context. %s", err))
	}
	r.loginOptions = LoginOptions{
		originContext:   originContext,
		isWCLogin:       len(r.flag.WCName) > 0,
		selfContainedMC: len(r.flag.SelfContained) > 0 && !(len(r.flag.WCName) > 0),
		selfContainedWC: len(r.flag.SelfContained) > 0 && len(r.flag.WCName) > 0,
	}
	r.loginOptions.switchToMCcontext = r.loginOptions.isWCLogin || !(r.loginOptions.selfContainedMC || r.flag.KeepContext)
	r.loginOptions.switchToWCcontext = r.loginOptions.isWCLogin && !(r.loginOptions.selfContainedWC || r.flag.KeepContext)
}

func (r *runner) tryToReuseExistingContext(ctx context.Context) error {
	config, err := r.k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	currentContext := config.CurrentContext
	kubeContextType := kubeconfig.GetKubeContextType(currentContext)

	switch kubeContextType {
	case kubeconfig.ContextTypeMC:
		authType := kubeconfig.GetAuthType(config, currentContext)
		if authType == kubeconfig.AuthTypeAuthProvider {
			authProvider, exists := kubeconfig.GetAuthProvider(config, currentContext)
			if !exists {
				return microerror.Maskf(incorrectConfigurationError, "There is no authentication configuration for the '%s' context", currentContext)
			}

			err = validateOIDCProvider(authProvider)
			if IsNewLoginRequired(err) {
				issuer := authProvider.Config[Issuer]

				err = r.loginWithURL(ctx, issuer, false, "")
				if err != nil {
					return microerror.Mask(err)
				}

				return nil
			} else if err != nil {
				return microerror.Maskf(incorrectConfigurationError, "The authentication configuration is corrupted, please log in again using a URL.")
			}
		} else if authType == kubeconfig.AuthTypeUnknown {
			return microerror.Maskf(incorrectConfigurationError, "There is no authentication configuration for the '%s' context", currentContext)
		}

		codeName := kubeconfig.GetCodeNameFromKubeContext(currentContext)
		fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))

		if r.loginOptions.isWCLogin {
			err = r.createClusterClientCert(ctx)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		return nil

	case kubeconfig.ContextTypeWC:
		codeName := kubeconfig.GetCodeNameFromKubeContext(currentContext)
		clusterName := kubeconfig.GetClusterNameFromKubeContext(currentContext)
		fmt.Fprint(r.stdout, color.GreenString("You are logged in to the workload cluster '%s' of installation '%s'.\n", clusterName, codeName))

		return nil

	default:
		if currentContext != "" {
			return microerror.Maskf(selectedContextNonCompatibleError, "The current context '%s' does not seem to belong to a Giant Swarm management cluster.\nPlease run 'kubectl gs login --help' to find out how to log in to a particular management cluster.", currentContext)
		}

		return microerror.Maskf(selectedContextNonCompatibleError, "The current context does not seem to belong to a Giant Swarm management cluster.\nPlease run 'kubectl gs login --help' to find out how to log in to a particular management cluster.")
	}
}

// loginWithKubeContextName switches the active kubernetes context to
// the one specified.
func (r *runner) loginWithKubeContextName(ctx context.Context, contextName string) error {
	codeName := kubeconfig.GetCodeNameFromKubeContext(contextName)
	err := r.loginWithCodeName(ctx, codeName)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Fprint(r.stdout, color.YellowString("Note: No need to pass the '%s' prefix. 'kubectl gs login %s' works fine.\n", kubeconfig.ContextPrefix, codeName))
	return nil
}

// loginWithCodeName switches the active kubernetes context to
// one with the name derived from the installation code name.
func (r *runner) loginWithCodeName(ctx context.Context, codeName string) error {
	var contextAlreadySelected bool
	var newLoginRequired bool

	contextName := kubeconfig.GenerateKubeContextName(codeName)
	err := switchContext(ctx, r.k8sConfigAccess, contextName, r.loginOptions.switchToMCcontext)
	if IsContextAlreadySelected(err) {
		contextAlreadySelected = true
	} else if IsNewLoginRequired(err) || IsTokenRenewalFailed(err) {
		newLoginRequired = true
	} else if err != nil {
		return microerror.Mask(err)
	}

	if newLoginRequired || r.loginOptions.selfContainedMC {
		config, err := r.k8sConfigAccess.GetStartingConfig()
		if err != nil {
			return microerror.Mask(err)
		}

		authType := kubeconfig.GetAuthType(config, contextName)
		if authType == kubeconfig.AuthTypeAuthProvider {
			// If we get here, we are sure that the kubeconfig context exists.
			authProvider, _ := kubeconfig.GetAuthProvider(config, contextName)
			issuer := authProvider.Config[Issuer]

			err = r.loginWithURL(ctx, issuer, false, "")
			if err != nil {
				return microerror.Mask(err)
			}
		}

		return nil
	}

	if contextAlreadySelected {
		_, _ = fmt.Fprintf(r.stdout, "Context '%s' is already selected.\n", contextName)
	} else if !r.loginOptions.isWCLogin && r.loginOptions.switchToMCcontext {
		_, _ = fmt.Fprintf(r.stdout, "Switched to context '%s'.\n", contextName)
	}

	_, _ = fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))

	return nil
}

// loginWithURL performs the OIDC login into an installation's
// k8s api with a happa/k8s api URL.
func (r *runner) loginWithURL(ctx context.Context, path string, firstLogin bool, tokenOverride string) error {
	athenaClient, err := graphql.NewClient(graphql.ClientImplConfig{
		HttpClient: http.DefaultClient,
		Url:        path,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	installationService, err := installation.New(installation.Config{
		AthenaClient: athenaClient,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	installationInfo, err := installationService.GetInfo(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if installation.GetUrlType(path) == installation.UrlTypeHappa {
		_, _ = fmt.Fprint(r.stdout, color.YellowString("Note: deriving Management API URL from web UI URL: %s\n", installationInfo.K8sApiURL))
	}

	var authResult authInfo
	{
		if len(tokenOverride) > 0 {
			authResult = authInfo{
				username: "automation",
				token:    tokenOverride,
			}
		} else {
			authResult, err = handleOIDC(ctx, r.stdout, r.stderr, installationInfo, r.flag.ClusterAdmin, r.flag.CallbackServerPort)
			if err != nil {
				return microerror.Mask(err)
			}

		}
	}
	if r.loginOptions.selfContainedMC {
		err = printMCCredentials(r.k8sConfigAccess, installationInfo, authResult, r.fs, r.flag.InternalAPI, r.flag.SelfContained)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		// Store kubeconfig and CA certificate.
		err = storeMCCredentials(r.k8sConfigAccess, installationInfo, authResult, r.fs, r.flag.InternalAPI, r.loginOptions.switchToMCcontext)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if len(authResult.email) > 0 {
		fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.email, installationInfo.Codename))
	} else {
		fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.username, installationInfo.Codename))
	}

	contextName := kubeconfig.GenerateKubeContextName(installationInfo.Codename)
	if r.loginOptions.selfContainedMC {
		fmt.Fprintf(r.stdout, "A new kubectl context has '%s' been created and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else {
		if firstLogin {
			if !r.loginOptions.switchToMCcontext {
				fmt.Fprintf(r.stdout, "A new kubectl context '%s' has been created.", contextName)
				fmt.Fprintf(r.stdout, " ")
			} else {
				fmt.Fprintf(r.stdout, "A new kubectl context '%s' has been created and selected.", contextName)
				fmt.Fprintf(r.stdout, " ")
			}
		}

		if !r.loginOptions.switchToMCcontext {
			fmt.Fprintf(r.stdout, "To switch to this context later, use either of these commands:\n\n")
		} else {
			fmt.Fprintf(r.stdout, "To switch back to this context later, use either of these commands:\n\n")

		}
		fmt.Fprintf(r.stdout, "  kubectl gs login %s\n", installationInfo.Codename)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	}
	return nil
}

func (r *runner) createClusterClientCert(ctx context.Context) error {
	config, err := commonconfig.New(commonconfig.CommonConfigConfig{
		ClientGetter: r.flag.config,
		Logger:       r.logger,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	provider, err := config.GetProvider(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	err = validateProvider(provider)
	if err != nil {
		return microerror.Mask(err)
	}

	var client k8sclient.Interface
	{
		client, err = config.GetClient()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var clientCertService clientcert.Interface
	{
		serviceConfig := clientcert.Config{
			Client: client.CtrlClient(),
		}
		clientCertService, err = clientcert.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var organizationService organization.Interface
	{
		serviceConfig := organization.Config{
			Client: client.CtrlClient(),
		}
		organizationService, err = organization.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var clusterService cluster.Interface
	{
		serviceConfig := cluster.Config{
			Client: client.CtrlClient(),
		}
		clusterService = cluster.New(serviceConfig)
	}

	var releaseService release.Interface
	{
		serviceConfig := release.Config{
			Client: client.CtrlClient(),
		}
		releaseService, err = release.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var c *cluster.Cluster
	{
		var namespaces []string
		if len(r.flag.WCOrganization) > 0 {
			orgNamespace, err := getOrganizationNamespace(ctx, organizationService, r.flag.WCOrganization)
			if err != nil {
				return microerror.Mask(err)
			}

			namespaces = append(namespaces, orgNamespace)
		} else {
			namespaces, err = getAllOrganizationNamespaces(ctx, organizationService)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		if r.flag.WCInsecureNamespace {
			namespaces = append(namespaces, "default")
		}

		c, err = findCluster(ctx, clusterService, organizationService, provider, r.flag.WCName, namespaces...)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	releaseVersion, err := getClusterReleaseVersion(c, provider, r.flag.WCInsecureNamespace)
	if err != nil {
		return microerror.Mask(err)
	}

	certOperatorVersion, err := getCertOperatorVersion(ctx, releaseService, releaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterBasePath, err := getClusterBasePath(r.k8sConfigAccess)
	if err != nil {
		return microerror.Mask(err)
	}

	certConfig := clientCertConfig{
		provider:            provider,
		clusterName:         r.flag.WCName,
		clusterNamespace:    c.Cluster.GetNamespace(),
		organizationName:    r.flag.WCOrganization,
		ttl:                 r.flag.WCCertTTL,
		groups:              r.flag.WCCertGroups,
		clusterBasePath:     clusterBasePath,
		certOperatorVersion: certOperatorVersion,
	}

	clientCertResource, err := createCert(ctx, clientCertService, certConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	// Retrieve client certificate credential.
	secret, err := fetchCredential(ctx, provider, clientCertService, clientCertResource)
	if err != nil {
		return microerror.Mask(err)
	}

	// Store client certificate credential either into the current kubeconfig or a self-contained file if a path is given.
	var contextExists bool
	var contextName string
	if r.loginOptions.selfContainedWC {
		contextName, contextExists, err = printWCCredentials(r.k8sConfigAccess, r.fs, r.flag.SelfContained, clientCertResource, secret, clusterBasePath, r.loginOptions)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		contextName, contextExists, err = storeWCCredentials(r.k8sConfigAccess, r.fs, clientCertResource, secret, clusterBasePath, r.loginOptions)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Cleaning up leftover resources.
	err = cleanUpClientCertResources(ctx, clientCertService, clientCertResource)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprint(r.stdout, color.GreenString("\nCreated client certificate for workload cluster '%s'.\n", r.flag.WCName))

	if r.loginOptions.selfContainedWC {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else if !r.loginOptions.switchToWCcontext {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s'. To switch back to this context later, use this command:\n\n", contextName)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	} else if contextExists {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n\n", contextName)
		fmt.Fprintf(r.stdout, "To switch back to this context later, use this command:\n\n")
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	} else {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected. To switch back to this context later, use this command:\n\n", contextName)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	}

	return nil
}

func getClusterBasePath(k8sConfigAccess clientcmd.ConfigAccess) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	clusterServer, _ := kubeconfig.GetClusterServer(config, config.CurrentContext)

	// Ensure any trailing ports are trimmed.
	reg := regexp.MustCompile(`:[0-9]+$`)
	clusterServer = reg.ReplaceAllString(clusterServer, "")

	// Some management clusters might have 'api.g8s' as prefix (example: Viking).
	clusterServer = strings.TrimPrefix(clusterServer, "https://api.g8s.")

	return strings.TrimPrefix(clusterServer, "https://g8s."), nil
}
