package client

import (
	applicationv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1"
	backupv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/backup/v1alpha1"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	examplev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/example/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	securityv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/security/v1alpha1"
	toolingv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/tooling/v1alpha1"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/client-go/rest"
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
		return nil, microerror.Maskf(InvalidConfigError, "%T.K8sRestConfig must not be empty", config)
	}

	var err error

	var k8sClient k8sclient.Interface
	{
		c := k8sclient.ClientsConfig{
			SchemeBuilder: k8sclient.SchemeBuilder{
				applicationv1alpha1.AddToScheme,
				backupv1alpha1.AddToScheme,
				corev1alpha1.AddToScheme,
				examplev1alpha1.AddToScheme,
				infrastructurev1alpha2.AddToScheme,
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
