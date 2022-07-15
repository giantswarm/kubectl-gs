package validate

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/cmd/validate/apps"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
)

const (
	name        = "validate"
	description = "Validate App CRs against a spec"
)

type Config struct {
	Logger       micrologger.Logger
	CommonConfig *commonconfig.CommonConfig

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.CommonConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CommonConfig must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	var err error

	var appsCmd *cobra.Command
	{
		c := apps.Config{
			Logger:       config.Logger,
			CommonConfig: config.CommonConfig,
			Stderr:       config.Stderr,
			Stdout:       config.Stdout,
		}

		appsCmd, err = apps.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	f := &flag{}

	r := &runner{
		commonConfig: config.CommonConfig,
		flag:         f,
		logger:       config.Logger,
		stderr:       config.Stderr,
		stdout:       config.Stdout,
	}

	c := &cobra.Command{
		Use:   name,
		Short: description,
		Long:  description,
		RunE:  r.Run,
	}

	f.Init(c)

	c.AddCommand(appsCmd)

	return c, nil
}
