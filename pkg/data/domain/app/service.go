package app

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app-operator/v2/service/controller/app/values"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

type Config struct {
	Client *client.Client
	Logger micrologger.Logger
}

type Service struct {
	client        *client.Client
	valuesService *values.Values

	catalogFetchResults map[string]CatalogFetchResult
	schemaFetchResults  map[string]SchemaFetchResult
}

func New(config Config) (Interface, error) {
	var err error

	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	var valuesService *values.Values
	{
		c := values.Config{
			K8sClient: config.Client.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		valuesService, err = values.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	s := &Service{
		client:              config.Client,
		valuesService:       valuesService,
		catalogFetchResults: make(map[string]CatalogFetchResult),
		schemaFetchResults:  make(map[string]SchemaFetchResult),
	}

	return s, nil
}
