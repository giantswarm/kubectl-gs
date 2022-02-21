package cluster

import "sigs.k8s.io/controller-runtime/pkg/client"

type Service struct {
	client client.Client
}

type Config struct {
	Client client.Client
}

func New(config Config) *Service {
	return &Service{
		client: config.Client,
	}
}
