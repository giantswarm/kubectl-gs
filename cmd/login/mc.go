package login

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
)

func (r *runner) handleMCLogin(ctx context.Context, installationIdentifier string) error {
	if _, contextType := kubeconfig.IsKubeContext(installationIdentifier); contextType == kubeconfig.ContextTypeMC || r.flag.EnforceContext {
		return r.loginWithKubeContextName(ctx, installationIdentifier)

	} else if kubeconfig.IsCodeName(installationIdentifier) {
		return r.loginWithCodeName(ctx, installationIdentifier)

	} else {
		var tokenOverride string
		if c, ok := r.flag.config.(*genericclioptions.ConfigFlags); ok && c.BearerToken != nil && len(*c.BearerToken) > 0 {
			tokenOverride = *c.BearerToken
		}
		return r.loginWithURL(ctx, installationIdentifier, true, tokenOverride)
	}
}

// loginWithKubeContextName switches the active kubernetes context to
// the one specified.
func (r *runner) loginWithKubeContextName(ctx context.Context, contextName string) error {
	var contextAlreadySelected bool
	var newLoginRequired bool

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
		fmt.Fprintf(r.stdout, "Context '%s' is already selected.\n", contextName)
	} else if !r.loginOptions.isWCLogin && r.loginOptions.switchToMCcontext {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n", contextName)
	}

	return nil
}

// loginWithCodeName switches the active kubernetes context to
// one with the name derived from the installation code name.
func (r *runner) loginWithCodeName(ctx context.Context, codeName string) error {
	contextName := kubeconfig.GenerateKubeContextName(codeName)
	err := r.loginWithKubeContextName(ctx, contextName)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Fprint(r.stdout, color.GreenString("You are logged in to the management cluster of installation '%s'.\n", codeName))
	return nil
}

// loginWithURL performs the OIDC login into an installation's
// k8s api with a happa/k8s api URL.
func (r *runner) loginWithURL(ctx context.Context, path string, firstLogin bool, tokenOverride string) error {
	i, err := installation.New(ctx, path)
	if installation.IsUnknownUrlType(err) {
		return microerror.Maskf(unknownUrlError, "'%s' is not a valid Giant Swarm Management API URL. Please check the spelling.\nIf not sure, pass the web UI URL of the installation or the installation handle as an argument instead.", path)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if installation.GetUrlType(path) == installation.UrlTypeHappa {
		fmt.Fprint(r.stdout, color.YellowString("Note: deriving Management API URL from web UI URL: %s\n", i.K8sApiURL))
	}
	err = r.loginWithInstallation(ctx, tokenOverride, i)
	if err != nil {
		return microerror.Mask(err)
	}

	contextName := kubeconfig.GenerateKubeContextName(i.Codename)
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
		fmt.Fprintf(r.stdout, "  kubectl gs login %s\n", i.Codename)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	}
	return nil

}

func (r *runner) loginWithInstallation(ctx context.Context, tokenOverride string, i *installation.Installation) error {
	var err error

	var authResult authInfo
	{
		if len(tokenOverride) > 0 {
			authResult = authInfo{
				username: "automation",
				token:    tokenOverride,
			}
		} else {
			authResult, err = handleOIDC(ctx, r.stdout, r.stderr, i, r.flag.ClusterAdmin, r.flag.CallbackServerPort)
			if err != nil {
				return microerror.Mask(err)
			}

		}
	}
	if r.loginOptions.selfContainedMC {
		err = printMCCredentials(r.k8sConfigAccess, i, authResult, r.fs, r.flag.InternalAPI, r.flag.SelfContained)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		// Store kubeconfig and CA certificate.
		err = storeMCCredentials(r.k8sConfigAccess, i, authResult, r.fs, r.flag.InternalAPI, r.loginOptions.switchToMCcontext)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if len(authResult.email) > 0 {
		fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.email, i.Codename))
	} else {
		fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.username, i.Codename))
	}
	return nil
}
