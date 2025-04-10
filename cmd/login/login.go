package login

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/v5/pkg/installation"
	"github.com/giantswarm/kubectl-gs/v5/pkg/kubeconfig"
)

func (r *runner) findContext(ctx context.Context, installationIdentifier string) (bool, error) {
	if _, contextType := kubeconfig.IsKubeContext(installationIdentifier); contextType != kubeconfig.ContextTypeNone {
		return true, r.loginWithKubeContextName(ctx, installationIdentifier)

	} else if kubeconfig.IsCodeName(installationIdentifier) || kubeconfig.IsWCCodeName(installationIdentifier) {
		return true, r.loginWithCodeName(ctx, installationIdentifier)

	}
	return false, nil
}

// loginWithKubeContextName switches the active kubernetes context to
// the one specified.
func (r *runner) loginWithKubeContextName(ctx context.Context, contextName string) error {
	var contextAlreadySelected bool
	var newLoginRequired bool
	k8sConfigAccess := r.commonConfig.GetConfigAccess()

	err := switchContext(ctx, k8sConfigAccess, contextName, r.loginOptions.switchToContext)
	if IsContextAlreadySelected(err) {
		contextAlreadySelected = true
	} else if IsNewLoginRequired(err) || IsTokenRenewalFailed(err) {
		newLoginRequired = true
	} else if err != nil {
		return microerror.Mask(err)
	}

	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	if newLoginRequired || r.loginOptions.selfContained {
		authType := kubeconfig.GetAuthType(config, contextName)
		if authType == kubeconfig.AuthTypeAuthProvider {
			// If we get here, we are sure that the kubeconfig context exists.
			server, _ := kubeconfig.GetClusterServer(config, contextName)

			err = r.loginWithURL(ctx, server, false, "")
			if err != nil {
				return microerror.Mask(err)
			}
		}

		return nil
	}

	if r.flag.DeviceAuth {
		_, _ = fmt.Fprintf(r.stdout, color.YellowString("A valid `%s` context already exists, there is no need to log in again, ignoring the `device-flow` flag.\n\n"), contextName)
		if clusterServer, exists := kubeconfig.GetClusterServer(config, contextName); exists {
			_, _ = fmt.Fprintf(r.stdout, "Run kubectl gs login with `%s` instead of the context name to force the re-login.\n", clusterServer)
		} else {
			_, _ = fmt.Fprintln(r.stdout, "Run kubectl gs login with the API URL to force the re-login.")
		}
	}

	if contextAlreadySelected {
		_, _ = fmt.Fprintf(r.stdout, "Context '%s' is already selected.\n", contextName)
	} else if !r.loginOptions.isWC && r.loginOptions.switchToContext {
		_, _ = fmt.Fprintf(r.stdout, "Switched to context '%s'.\n", contextName)
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
	_, _ = fmt.Fprint(r.stdout, color.GreenString("You are logged in to the cluster '%s'.\n", codeName))
	return nil
}

// loginWithURL performs the OIDC login into an installation's
// k8s api with a happa/k8s api URL.
func (r *runner) loginWithURL(ctx context.Context, path string, firstLogin bool, tokenOverride string) error {
	i, err := r.commonConfig.GetInstallation(ctx, path, "")
	if installation.IsUnknownUrlType(err) {
		return microerror.Maskf(unknownUrlError, "'%s' is not a valid Giant Swarm Management API URL. Please check the spelling.\nIf not sure, pass the web UI URL of the installation or the installation handle as an argument instead.", path)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if installation.GetUrlType(path) == installation.UrlTypeHappa {
		_, _ = fmt.Fprint(r.stdout, color.YellowString("Note: deriving Management API URL from web UI URL: %s\n", i.K8sApiURL))
	}
	err = r.loginWithInstallation(ctx, tokenOverride, i)
	if err != nil {
		return microerror.Mask(err)
	}

	contextName := kubeconfig.GenerateKubeContextName(i.Codename)
	if r.loginOptions.selfContained {
		_, _ = fmt.Fprintf(r.stdout, "A new kubectl context has '%s' been created and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		_, _ = fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else {
		if firstLogin {
			if !r.loginOptions.switchToContext {
				_, _ = fmt.Fprintf(r.stdout, "A new kubectl context '%s' has been created.", contextName)
				_, _ = fmt.Fprintf(r.stdout, " ")
			} else {
				_, _ = fmt.Fprintf(r.stdout, "A new kubectl context '%s' has been created and selected.", contextName)
				_, _ = fmt.Fprintf(r.stdout, " ")
			}
		}

		if !r.loginOptions.switchToContext {
			_, _ = fmt.Fprintf(r.stdout, "To switch to this context later, use either of these commands:\n\n")
		} else {
			_, _ = fmt.Fprintf(r.stdout, "To switch back to this context later, use either of these commands:\n\n")

		}
		_, _ = fmt.Fprintf(r.stdout, "  kubectl gs login %s\n", i.Codename)
		_, _ = fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	}
	return nil

}

func (r *runner) loginWithInstallation(ctx context.Context, tokenOverride string, i *installation.Installation) error {
	k8sConfigAccess := r.commonConfig.GetConfigAccess()

	var err error
	var authResult authInfo
	{
		if len(tokenOverride) > 0 {
			authResult = authInfo{
				username: "automation",
				token:    tokenOverride,
			}
		} else {
			contextName := kubeconfig.GenerateKubeContextName(i.Codename)
			if r.flag.DeviceAuth || r.isDeviceAuthContext(k8sConfigAccess, contextName) {
				authResult, err = handleDeviceFlowOIDC(r.stdout, r.stderr, i, r.flag.InternalAPI)
			} else {
				authResult, err = handleOIDC(ctx, r.stdout, r.stderr, i, r.flag.ConnectorID, r.flag.ClusterAdmin, r.flag.InternalAPI, r.flag.CallbackServerHost, r.flag.CallbackServerPort, r.flag.LoginTimeout)
				if err != nil && errors.Is(err, context.DeadlineExceeded) || IsAuthResponseTimedOut(err) {
					_, _ = fmt.Fprintf(r.stderr, "\nYour authentication flow timed out after %s. Please execute the same command again.\n", r.flag.LoginTimeout.String())
					_, _ = fmt.Fprintf(r.stderr, "You can use the --login-timeout flag to configure a longer timeout interval, for example --login-timeout=%.0fs.\n", 2*r.flag.LoginTimeout.Seconds())
					if errors.Is(err, context.DeadlineExceeded) {
						return microerror.Maskf(authResponseTimedOutError, "failed to get an authentication response on time")
					}
				}
			}
			if err != nil {
				return microerror.Mask(err)
			}

		}
	}
	if r.loginOptions.selfContained {
		err = printMCCredentials(k8sConfigAccess, i, authResult, r.fs, r.flag.InternalAPI, r.flag.SelfContained)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		// Store kubeconfig and CA certificate.
		err = storeMCCredentials(k8sConfigAccess, i, authResult, r.flag.InternalAPI, r.loginOptions.switchToContext)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if len(authResult.email) > 0 {
		_, _ = fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on cluster '%s'.\n\n", authResult.email, i.Codename))
	} else {
		_, _ = fmt.Fprint(r.stdout, color.GreenString("Logged in successfully as '%s' on cluster '%s'.\n\n", authResult.username, i.Codename))
	}
	return nil
}

func (r *runner) isDeviceAuthContext(k8sConfigAccess clientcmd.ConfigAccess, contextName string) bool {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return false
	}

	if originContext, ok := config.Contexts[contextName]; ok {
		return isDeviceAuthInfo(originContext.AuthInfo)
	}

	return false
}

func isDeviceAuthInfo(authInfo string) bool {
	return strings.HasSuffix(authInfo, "-device")
}
