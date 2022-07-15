package nodepool

import (
	"io"
	"os"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

const (
	name        = "nodepool"
	description = "Template Giant Swarm Node Pool CRs."
)

type Config struct {
	CommonConfig *commonconfig.CommonConfig
	Logger       micrologger.Logger
	Stderr       io.Writer
	Stdout       io.Writer
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

	return c, nil
}
