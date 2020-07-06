package login

import (
	"context"
	"fmt"
	"io"

	gooidc "github.com/coreos/go-oidc"
	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
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

	authResult, err := handleAuth(ctx, r.stdout, i)
	if err != nil {
		return microerror.Mask(err)
	}

	// Store kubeconfig and CA certificate.
	err = storeCredentials(r.k8sConfigAccess, i, authResult, r.fs)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintf(r.stdout, color.GreenString("Logged in successfully as '%s' on installation '%s'.\n\n", authResult.Email, i.Codename))

	contextName := generateKubeContextName(i.Codename)
	fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected.", contextName)
	fmt.Fprintf(r.stdout, " ")
	fmt.Fprintf(r.stdout, "To switch back to this context later, use either of these commands:\n\n")
	fmt.Fprintf(r.stdout, "  kgs login %s\n", i.Codename)
	fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)

	return nil
}
