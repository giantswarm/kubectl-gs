package appcatalog

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/microerror"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid appcatalog getter Service.
type Config struct {
	Client *client.Client
}

// Service is the object we'll hang the appcatalog getter methods on.
type Service struct {
	client *client.Client
}

// New returns a new appcatalog getter Service.
func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

// Get fetches an AppCatalog CR by name.
func (s *Service) Get(ctx context.Context, options GetOptions) (*applicationv1alpha1.AppCatalog, error) {
	objKey := runtimeClient.ObjectKey{
		Name: options.Name,
	}

	appcatalog := &applicationv1alpha1.AppCatalog{}
	err := s.client.K8sClient.CtrlClient().Get(ctx, objKey, appcatalog)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return appcatalog, nil
}
