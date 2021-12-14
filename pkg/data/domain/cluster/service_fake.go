package cluster

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &FakeService{}

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

func (ms *FakeService) Patch(ctx context.Context, object runtime.Object, options PatchOptions) error {
	var err error

	bytes, err := json.Marshal(options.PatchSpecs)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ms.service.client.K8sClient.CtrlClient().Patch(ctx, object, runtimeclient.RawPatch(types.JSONPatchType, bytes))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
