package credentialplugin

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

const (
	name        = "credential-plugin"
	description = "OIDC credential plugin for kubectl exec authentication"
)

type Config struct {
	Logger micrologger.Logger

	Stderr io.Writer
	Stdout io.Writer
	Stdin  io.Reader
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}
	if config.Stdin == nil {
		config.Stdin = os.Stdin
	}

	r := &runner{
		stderr: config.Stderr,
		stdout: config.Stdout,
		stdin:  config.Stdin,
	}

	c := &cobra.Command{
		Use:    name,
		Short:  description,
		Long:   description,
		Hidden: true, // Hide from help as this is an internal command
		RunE:   r.Run,
	}

	return c, nil
}
