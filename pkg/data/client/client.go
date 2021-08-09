package client

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	backupv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/backup/v1alpha1"
	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	examplev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/example/v1alpha1"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	toolingv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/tooling/v1alpha1"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/rest"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
)

type Config struct {
	Logger micrologger.Logger

	K8sRestConfig *rest.Config
}

type Client struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

func New(config Config) (*Client, error) {
	if config.K8sRestConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sRestConfig must not be empty", config)
	}

	var err error

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			SchemeBuilder: k8sclient.SchemeBuilder{
				capiv1alpha3.AddToScheme,
				capzv1alpha3.AddToScheme,
				capiexpv1alpha3.AddToScheme,
				capzexpv1alpha3.AddToScheme,
				applicationv1alpha1.AddToScheme,
				backupv1alpha1.AddToScheme,
				corev1alpha1.AddToScheme,
				examplev1alpha1.AddToScheme,
				infrastructurev1alpha3.AddToScheme,
				providerv1alpha1.AddToScheme,
				releasev1alpha1.AddToScheme,
				securityv1alpha1.AddToScheme,
				toolingv1alpha1.AddToScheme,
			},
			Logger:     config.Logger,
			RestConfig: config.K8sRestConfig,
		}

		k8sClient, err = k8sclient.NewClients(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &Client{
		K8sClient: k8sClient,
		Logger:    config.Logger,
	}

	return c, nil
}
