package nodepool

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type GetOptions struct {
	ID        string
	Provider  string
	Namespace string
}

type Interface interface {
	Get(context.Context, GetOptions) (runtime.Object, error)
}
