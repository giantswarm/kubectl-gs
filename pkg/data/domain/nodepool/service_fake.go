package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dataclient "github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &FakeService{}

type FakeService struct {
	service *Service
	storage []client.Object
}

func NewFakeService(storage []client.Object) *FakeService {
	clientConfig := dataclient.Config{
		Logger: microloggertest.New(),
	}
	fakeClient, _ := dataclient.NewFakeClient(clientConfig)

	underlyingService := &Service{
		client: fakeClient,
	}

	ms := &FakeService{
		service: underlyingService,
		storage: storage,
	}

	return ms
}

func (ms *FakeService) Get(ctx context.Context, options GetOptions) (Resource, error) {
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
