package logout

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	idTokenKey      = "id-token"
	refreshTokenKey = "refresh-token"
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
	var refreshTokenDeleted bool
	var userEntryName string
	var currentContextName string

	{
		// Determine current context
		config, err := r.k8sConfigAccess.GetStartingConfig()
		if err != nil {
			return microerror.Mask(err)
		}
		if config == nil {
			return microerror.Mask(noKubeconfig)
		}

		currentContextName = config.CurrentContext
		if currentContextName == "" {
			return microerror.Mask(noKubeconfigCurrentContext)
		}

		context, ok := config.Contexts[currentContextName]
		if !ok {
			return microerror.Mask(noKubeconfigCurrentContext)
		}

		userEntryName = context.AuthInfo
		if userEntryName == "" {
			return microerror.Mask(noKubeconfigCurrentUser)
		}

		_, ok = config.AuthInfos[userEntryName]
		if !ok {
			return microerror.Mask(noKubeconfigCurrentUser)
		}

		_, ok2 := config.AuthInfos[userEntryName].AuthProvider.Config[idTokenKey]
		if !ok2 {
			return microerror.Mask(noIDToken)
		}

		// Delete value
		delete(config.AuthInfos[userEntryName].AuthProvider.Config, idTokenKey)

		if !r.flag.KeepRefreshToken {
			_, ok3 := config.AuthInfos[userEntryName].AuthProvider.Config[refreshTokenKey]
			if !ok3 {
				return microerror.Mask(noRefreshToken)
			}

			// Delete value
			delete(config.AuthInfos[userEntryName].AuthProvider.Config, refreshTokenKey)
			refreshTokenDeleted = true
		}

		err = clientcmd.ModifyConfig(r.k8sConfigAccess, *config, false)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	fmt.Fprintf(r.stdout, "Logging out user %s in current context %s\n", userEntryName, currentContextName)
	if refreshTokenDeleted {
		fmt.Fprintf(r.stdout, "Deleting '%s' and '%s'.\n", idTokenKey, refreshTokenKey)
	} else {
		fmt.Fprintf(r.stdout, "Deleting '%s', kept '%s' in place.\n", idTokenKey, refreshTokenKey)
	}

	return nil
}
