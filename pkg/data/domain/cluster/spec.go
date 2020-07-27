package cluster

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type GetOptions struct {
	ID        string
	Provider  string
	Namespace string
}

// Interface represents the contract for the clusters service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Get(context.Context, *GetOptions) (runtime.Object, error)
}
