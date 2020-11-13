package app

import (
	"context"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
)

// GetOptions are the parameters that the Get method takes.
type GetOptions struct {
	Name      string
	Namespace string
}

// Interface represents the contract for the app data service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Get(context.Context, GetOptions) (*applicationv1alpha1.App, error)
}
