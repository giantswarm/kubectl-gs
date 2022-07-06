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
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"

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
	isWCClientCert            bool
	selfContained             bool
	selfContainedClientCert   bool
	switchToContext           bool
	switchToClientCertContext bool
	originContext             string
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	r.setLoginOptions(ctx, &args)
	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	switch len(args) {
	// No arguments given - we try to reuse the existing context.
	case 0:
		err := r.tryToReuseExistingContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	// One argument - This can be an existing context, an installation name or a login URL
	case 1:
		installationIdentifier := strings.ToLower(args[0])
		foundContext, err := r.findContext(ctx, installationIdentifier)
		if IsContextDoesNotExist(err) && !strings.HasSuffix(installationIdentifier, kubeconfig.ClientCertSuffix) {
			clientCertContext := kubeconfig.GetClientCertContextName(installationIdentifier)
			fmt.Fprint(r.stdout, color.YellowString("No context named %s was found: %s\nLooking for context %s.\n", installationIdentifier, err, clientCertContext))
			foundContext, err = r.findContext(ctx, clientCertContext)
		}
		if err != nil {
			return microerror.Mask(err)
		}
		if !foundContext {
			var tokenOverride string
			if c, ok := r.flag.config.(*genericclioptions.ConfigFlags); ok && c.BearerToken != nil && len(*c.BearerToken) > 0 {
				tokenOverride = *c.BearerToken
			}
			err = r.loginWithURL(ctx, installationIdentifier, true, tokenOverride)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	// Two arguments - This is interpreted as an installation and a workload cluster
	case 2:
		installationIdentifier := strings.ToLower(strings.Join(args, "-"))
		foundContext, err := r.findContext(ctx, installationIdentifier)
		if IsContextDoesNotExist(err) && !strings.HasSuffix(installationIdentifier, kubeconfig.ClientCertSuffix) {
			clientCertContext := kubeconfig.GetClientCertContextName(installationIdentifier)
			fmt.Fprint(r.stdout, color.YellowString("No context named %s was found: %s\nLooking for context %s.\n", installationIdentifier, err, clientCertContext))
			foundContext, err = r.findContext(ctx, clientCertContext)
		}
		if err != nil {
			return microerror.Mask(err)
		}
		if !foundContext {
			return microerror.Maskf(contextDoesNotExistError, "Could not find context for identifier %s", installationIdentifier)
		}
	default:
		return microerror.Maskf(invalidConfigError, "Invalid number of arguments.")
	}

	// Clientcert creation if desired
	if r.loginOptions.isWCClientCert {
		return r.handleWCClientCert(ctx)
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

func (r *runner) tryToGetContextFlag(ctx context.Context) string {
	config, ok := r.flag.config.(*genericclioptions.ConfigFlags)
	if !ok {
		return ""
	}
	return *config.Context
}

func (r *runner) setLoginOptions(ctx context.Context, args *[]string) {
	originContext, err := r.tryToGetCurrentContext(ctx)
	if err != nil {
		fmt.Fprintln(r.stdout, color.YellowString("Failed trying to determine current context. %s", err))
	}
	contextFlag := r.tryToGetContextFlag(ctx)
	if contextFlag != "" && len(*args) < 1 {
		*args = append(*args, contextFlag)
	}
	r.loginOptions = LoginOptions{
		originContext:           originContext,
		isWCClientCert:          len(r.flag.WCName) > 0,
		selfContained:           len(r.flag.SelfContained) > 0 && !(len(r.flag.WCName) > 0),
		selfContainedClientCert: len(r.flag.SelfContained) > 0 && len(r.flag.WCName) > 0,
	}
	r.loginOptions.switchToContext = r.loginOptions.isWCClientCert || !(r.loginOptions.selfContained || r.flag.KeepContext)
	r.loginOptions.switchToClientCertContext = r.loginOptions.isWCClientCert && !(r.loginOptions.selfContainedClientCert || r.flag.KeepContext)
}

func (r *runner) tryToReuseExistingContext(ctx context.Context) error {
	config, err := r.k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return microerror.Mask(err)
	}
	currentContext := config.CurrentContext
	if currentContext != "" {
		return r.loginWithKubeContextName(ctx, currentContext)
	} else {
		return microerror.Maskf(selectedContextNonCompatibleError, "The current context does not seem to belong to a Giant Swarm cluster.\nPlease run 'kubectl gs login --help' to find out how to log in to a particular cluster.")
	}
}
