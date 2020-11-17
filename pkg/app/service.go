package app

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/app/v3/pkg/values"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
	appdata "github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	appcatalogdata "github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalog"
)

var _ Interface = &Service{}

// Config represents the values that New() needs in order to create a valid app service.
type Config struct {
	Client *client.Client
	Logger micrologger.Logger
}

// Service represents an instance of the App service.
type Service struct {
	client                *client.Client
	appDataService        appdata.Interface
	appcatalogDataService appcatalogdata.Interface
	valuesService         *values.Values

	catalogFetchResults map[string]CatalogFetchResult
	schemaFetchResults  map[string]SchemaFetchResult
}

// New returns an app service given a certain Config.
func New(config Config) (Interface, error) {
	var err error

	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var appDataService appdata.Interface
	{
		c := appdata.Config{
			Client: config.Client,
		}

		appDataService, err = appdata.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var appcatalogDataService appcatalogdata.Interface
	{
		c := appcatalogdata.Config{
			Client: config.Client,
		}

		appcatalogDataService, err = appcatalogdata.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
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
		client:                config.Client,
		appDataService:        appDataService,
		appcatalogDataService: appcatalogDataService,
		valuesService:         valuesService,
		catalogFetchResults:   make(map[string]CatalogFetchResult),
		schemaFetchResults:    make(map[string]SchemaFetchResult),
	}

	return s, nil
}
