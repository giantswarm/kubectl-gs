package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

const (
	clusterNamespace = "default"
)

type Config struct {
	Client *client.Client
}

type Service struct {
	client *client.Client
}

func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(client.InvalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

func (s *Service) GetByID(ctx context.Context, id string, options *ListOptions) (*apiv1alpha2.Cluster, error) {
	cluster := &apiv1alpha2.Cluster{}
	key := runtimeClient.ObjectKey{
		Name:      id,
		Namespace: clusterNamespace,
	}
	err := s.client.K8sClient.CtrlClient().Get(ctx, key, cluster)
	if errors.IsNotFound(err) {
		return nil, microerror.Mask(notFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	// Adding these manually, because they're missing in the listed object.
	// cluster.TypeMeta = v1alpha1.NewAppCatalogTypeMeta()

	return cluster, nil
}
