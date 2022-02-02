package app

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
	catalogdata "github.com/giantswarm/kubectl-gs/pkg/data/domain/catalog"
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

	var fakeCatalogDataService catalogdata.Interface
	{
		c := catalogdata.Config{
			Client: fakeClient,
		}

		fakeCatalogDataService, _ = catalogdata.New(c)
	}

	underlyingService := &Service{
		client:             fakeClient,
		catalogDataService: fakeCatalogDataService,
	}

	fs := &FakeService{
		service: underlyingService,
		storage: storage,
	}

	return fs
}

func (fs *FakeService) Get(ctx context.Context, options GetOptions) (Resource, error) {
	var err error
	for _, res := range fs.storage {
		err = fs.service.client.K8sClient.CtrlClient().Create(ctx, res)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	result, err := fs.service.Get(ctx, options)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return result, nil
}

func (fs *FakeService) Patch(ctx context.Context, options PatchOptions) error {
	var err error
	for _, res := range fs.storage {
		err = fs.service.client.K8sClient.CtrlClient().Create(ctx, res)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = fs.service.Patch(ctx, options)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
