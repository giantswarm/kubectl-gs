// Package lkogout provides the 'logout' command which is used
// to remove credentials from the current kubeconfig context's user.
package logout

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
)

const (
	name             = "logout"
	shortDescription = "Logs out from an installation's Management API"
	longDescription  = `Logs out from an installation's Management API.

This removes credentials (secrets) from the current kubeconfig context, specifically
from the user entry associated with that context.

By default, both the ID token and the refresh token are removed. When setting
the --keep-refresh-token falg to true, only the ID token is removed.
`
)

type Config struct {
	Logger     micrologger.Logger
	FileSystem afero.Fs

	K8sConfigAccess clientcmd.ConfigAccess

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.FileSystem == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.FileSystem must not be empty", config)
	}
	if config.K8sConfigAccess == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigAccess must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		fs:     config.FileSystem,

		k8sConfigAccess: config.K8sConfigAccess,

		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:   name,
		Short: shortDescription,
		Long:  longDescription,
		RunE:  r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(config.K8sConfigAccess),
		),
	}

	f.Init(c)

	return c, nil
}
