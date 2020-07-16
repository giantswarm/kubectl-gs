package cluster

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

type FakeService struct {
	service *Service
	storage []runtime.Object
}

func NewFakeService(storage []runtime.Object) *FakeService {
	clientConfig := client.Config{
		Logger: microloggertest.New(),
	}
	fakeClient, _ := client.NewFakeClient(clientConfig)

	underlyingService := &Service{
		client: fakeClient,
	}

	ms := &FakeService{
		service: underlyingService,
		storage: storage,
	}

	return ms
}

func (ms *FakeService) Get(ctx context.Context, options *GetOptions) (runtime.Object, error) {
	var err error
	for _, res := range ms.storage {
		err = ms.service.client.K8sClient.CtrlClient().Create(ctx, res)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	result, err := ms.service.Get(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return result, nil
}
