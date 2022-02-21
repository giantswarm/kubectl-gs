package release

import (
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Interface = &Service{}

type Config struct {
	Client client.Client
}

type Service struct {
	client client.Client
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
