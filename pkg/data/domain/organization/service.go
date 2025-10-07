package organization

import (
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
)

var _ Interface = &Service{}

type Config struct {
	Client k8sclient.Interface
}

type Service struct {
	client k8sclient.Interface
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
