package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
)

func (s *Service) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var resource Resource
	var err error

	if len(options.Name) > 0 {
		resource, err = s.getByIdCAPI(ctx, options.Name, options.Namespace, options.ClusterName)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		resource, err = s.getAllCAPI(ctx, options.Namespace, options.ClusterName)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resource, nil
}
