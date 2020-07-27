package cluster

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

type Config struct {
	Client *client.Client
}

type Service struct {
	client *client.Client
}

func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}
