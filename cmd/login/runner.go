package login

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
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
	var err error

	isCreatingClientCert := len(r.flag.WCName) > 0

	if len(args) < 1 {
		err = r.tryToReuseExistingContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		if isCreatingClientCert {
			err = r.createClusterClientCert(ctx)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		return nil
	}

	// This can be a kubernetes context name,
	// installation code name, or happa/k8s api URL.
	installationIdentifier := strings.ToLower(args[0])

	switch {
	case kubeconfig.IsKubeContext(installationIdentifier):
		err = r.loginWithKubeContextName(ctx, installationIdentifier)
		if err != nil {
			return microerror.Mask(err)
		}

	case kubeconfig.IsCodeName(installationIdentifier):
		err = r.loginWithCodeName(ctx, installationIdentifier)
		if err != nil {
			return microerror.Mask(err)
		}

	default:
		err = r.loginWithURL(ctx, installationIdentifier, true)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if isCreatingClientCert {
		err = r.createClusterClientCert(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *runner) tryToReuseExistingContext(ctx context.Context) error {
	config, err := r.k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	currentContext, isLoggedInWithKubeContext := isLoggedWithGSContext(config)
	if isLoggedInWithKubeContext {
		authProvider, exists := kubeconfig.GetAuthProvider(config, currentContext)
		if !exists {
			return microerror.Maskf(incorrectConfigurationError, "There is no authentication configuration for the '%s' context", currentContext)
		}

		err = validateAuthProvider(authProvider)
		if IsNewLoginRequired(err) {
			issuer := authProvider.Config[Issuer]

			err = r.loginWithURL(ctx, issuer, false)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		} else if err != nil {
			return microerror.Maskf(incorrectConfigurationError, "The authentication configuration is corrupted, please log in again using a URL.")
		}

		codeName := kubeconfig.GetCodeNameFromKubeContext(currentContext)
		fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))

		return nil
	}

	if currentContext != "" {
		return microerror.Maskf(selectedContextNonCompatibleError, "The current context '%s' does not seem to belong to a Giant Swarm management cluster.\nPlease run 'kubectl gs login --help' to find out how to log in to a particular management cluster.", currentContext)
	}

	return microerror.Maskf(selectedContextNonCompatibleError, "The current context does not seem to belong to a Giant Swarm management cluster.\nPlease run 'kubectl gs login --help' to find out how to log in to a particular management cluster.")
}

// loginWithKubeContextName switches the active kubernetes context to
// the one specified.
func (r *runner) loginWithKubeContextName(ctx context.Context, contextName string) error {
	var contextAlreadySelected bool

	codeName := kubeconfig.GetCodeNameFromKubeContext(contextName)
	err := switchContext(ctx, r.k8sConfigAccess, contextName)
	if IsContextAlreadySelected(err) {
		contextAlreadySelected = true
	} else if IsNewLoginRequired(err) {
		config, err := r.k8sConfigAccess.GetStartingConfig()
		if err != nil {
			return microerror.Mask(err)
		}

		// If we get here, we are sure that the kubeconfig context exists.
		authProvider, _ := kubeconfig.GetAuthProvider(config, contextName)
		issuer := authProvider.Config[Issuer]

		err = r.loginWithURL(ctx, issuer, false)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprint(r.stdout, color.YellowString("Note: No need to pass the '%s' prefix. 'kubectl gs login %s' works fine.\n", kubeconfig.ContextPrefix, codeName))

	if contextAlreadySelected {
		fmt.Fprintf(r.stdout, "Context '%s' is already selected.\n", contextName)
	} else {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n", contextName)
	}

	fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))

	return nil
}

// loginWithCodeName switches the active kubernetes context to
// one with the name derived from the installation code name.
func (r *runner) loginWithCodeName(ctx context.Context, codeName string) error {
	var contextAlreadySelected bool

	contextName := kubeconfig.GenerateKubeContextName(codeName)
	err := switchContext(ctx, r.k8sConfigAccess, contextName)
	if IsContextAlreadySelected(err) {
		contextAlreadySelected = true
	} else if IsNewLoginRequired(err) {
		config, err := r.k8sConfigAccess.GetStartingConfig()
		if err != nil {
			return microerror.Mask(err)
		}

		// If we get here, we are sure that the kubeconfig context exists.
		authProvider, _ := kubeconfig.GetAuthProvider(config, contextName)
		issuer := authProvider.Config[Issuer]

		err = r.loginWithURL(ctx, issuer, false)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if contextAlreadySelected {
		fmt.Fprintf(r.stdout, "Context '%s' is already selected.\n", contextName)
	} else {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n", contextName)
	}

	fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))

	return nil
}

// loginWithURL performs the OIDC login into an installation's
// k8s api with a happa/k8s api URL.
func (r *runner) loginWithURL(ctx context.Context, path string, firstLogin bool) error {
	i, err := installation.New(ctx, path)
	if installation.IsUnknownUrlType(err) {
		return microerror.Maskf(unknownUrlError, "'%s' is not a valid Giant Swarm Management API URL. Please check the spelling.\nIf not sure, pass the web UI URL of the installation or the installation handle as an argument instead.", path)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if installation.GetUrlType(path) == installation.UrlTypeHappa {
		fmt.Fprint(r.stdout, color.YellowString("Note: deriving Management API URL from web UI URL: %s\n", i.K8sApiURL))
	}

	authResult, err := handleAuth(ctx, r.stdout, r.stderr, i, r.flag.ClusterAdmin, r.flag.CallbackServerPort)
	if err != nil {
		return microerror.Mask(err)
	}

	// Store kubeconfig and CA certificate.
	err = storeCredentials(r.k8sConfigAccess, i, authResult, r.fs, r.flag.InternalAPI)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.Email, i.Codename))

	contextName := kubeconfig.GenerateKubeContextName(i.Codename)
	if firstLogin {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected.", contextName)
		fmt.Fprintf(r.stdout, " ")
	}
	fmt.Fprintf(r.stdout, "To switch back to this context later, use either of these commands:\n\n")
	fmt.Fprintf(r.stdout, "  kubectl gs login %s\n", i.Codename)
	fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)

	return nil
}

func (r *runner) createClusterClientCert(ctx context.Context) error {
	var clientCertService clientcert.Interface
	{
		config := commonconfig.New(r.flag.config)

		client, err := config.GetClient(r.logger)
		if err != nil {
			return microerror.Mask(err)
		}

		serviceConfig := clientcert.Config{
			Client: client,
		}
		clientCertService, err = clientcert.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	clusterBasePath, err := getClusterBasePath(r.k8sConfigAccess)
	if err != nil {
		return microerror.Mask(err)
	}

	clientCertResource, err := generateClientCert(r.flag.WCName, r.flag.WCOrganization, r.flag.WCCertTTL, r.flag.WCCertGroups, clusterBasePath)
	if err != nil {
		return microerror.Mask(err)
	}

	// Creating client certificate resource.
	err = clientCertService.Create(ctx, clientCertResource)
	if err != nil {
		return microerror.Mask(err)
	}

	// Retrieving client certificate credential.
	secret, err := tryToGetClientCertCredential(ctx, clientCertService, clientCertResource.CertConfig.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	// Storing client certificate credential into the kubeconfig.
	contextName, contextExists, err := storeClientCertCredential(r.k8sConfigAccess, r.fs, clientCertResource, secret, clusterBasePath)
	if err != nil {
		return microerror.Mask(err)
	}

	// Cleaning up leftover resources.
	err = cleanUpClientCertResources(ctx, clientCertService, clientCertResource)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprint(r.stdout, color.GreenString("\nCreated client certificate for workload cluster '%s'.\n", r.flag.WCName))

	if contextExists {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n\n", contextName)
		fmt.Fprintf(r.stdout, "To switch back to this context later, use this command:\n\n")
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	} else {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected.\n", contextName)
	}

	return nil
}

func getClusterBasePath(k8sConfigAccess clientcmd.ConfigAccess) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}
	// If we get here, we are sure that the kubeconfig context exists and is valid.
	authProvider, _ := kubeconfig.GetAuthProvider(config, config.CurrentContext)

	issuer := authProvider.Config[Issuer]
	mcBasePath, err := installation.GetBasePath(issuer)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.TrimPrefix(mcBasePath, "g8s."), nil
}
