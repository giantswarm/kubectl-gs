package cluster

import (
	"context"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

type ListOptions struct {
	Provider string
}

type Interface interface {
	V4ListAzure(context.Context, *ListOptions) (*providerv1alpha1.AzureConfigList, error)
	V4ListAWS(context.Context, *ListOptions) (*providerv1alpha1.AWSConfigList, error)
	V5ListAWS(context.Context, *ListOptions) (*infrastructurev1alpha2.AWSClusterList, error)
	V4ListKVM(context.Context, *ListOptions) (*providerv1alpha1.KVMConfigList, error)
}
