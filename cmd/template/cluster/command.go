package cluster

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/pkg/middleware/renewtoken"
)

const (
	name        = "cluster"
	description = "Template Giant Swarm clusters."
)

type Config struct {
	Logger micrologger.Logger

	CommonConfig *commonconfig.CommonConfig

	Stderr io.Writer
	Stdout io.Writer
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
	if config.CommonConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CommonConfig must not be empty", config)
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
		PreRunE: middleware.Compose(
			renewtoken.Middleware(config.CommonConfig.ToRawKubeConfigLoader().ConfigAccess()),
		),
	}

	f.Init(c)

	return c, nil
}
