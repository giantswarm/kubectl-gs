package commonconfig

import (
	"context"
	"fmt"
	"net/http"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/pkg/graphql"
	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/scheme"
)

type CommonConfigConfig struct {
	ClientGetter genericclioptions.RESTClientGetter
	Logger       micrologger.Logger
}

type CommonConfig struct {
	clientGetter genericclioptions.RESTClientGetter
	httpClient   *http.Client
	logger       micrologger.Logger

	athenaClient        graphql.Client
	k8sClient           k8sclient.Interface
	installationService *installation.Service
}

func New(config CommonConfigConfig) (*CommonConfig, error) {
	if config.ClientGetter == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientGetter must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	return &CommonConfig{
		clientGetter: config.ClientGetter,
		logger:       config.Logger,
	}, nil
}

func (cc *CommonConfig) GetInstallationService(ctx context.Context) (*installation.Service, error) {
	if cc.installationService != nil {
		return cc.installationService, nil
	}

	athenaClient, err := cc.GetAthenaClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc.installationService, err = installation.New(installation.Config{
		AthenaClient: athenaClient,
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.installationService, nil
}

func (cc *CommonConfig) GetAthenaClient(ctx context.Context) (graphql.Client, error) {
	if cc.athenaClient != nil {
		return cc.athenaClient, nil
	}

	k8sClient, err := cc.GetClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var athenaIngress networking.Ingress
	err = k8sClient.CtrlClient().Get(ctx, client.ObjectKey{
		Namespace: "giantswarm",
		Name:      "athena",
	}, &athenaIngress)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var athenaIngressRule *networking.IngressRule
	for _, rule := range athenaIngress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.ServiceName == "athena" {
				athenaIngressRule = &rule
				break
			}
		}
		if athenaIngressRule != nil {
			break
		}
	}

	if athenaIngressRule == nil {
		return nil, microerror.Maskf(notFoundError, "ingress for athena service not found")
	}

	graphQLEndpoint := fmt.Sprintf("https://%s/graphql", athenaIngressRule.Host)
	cc.athenaClient, err = graphql.NewClient(graphql.ClientImplConfig{
		HttpClient: cc.httpClient,
		Url:        graphQLEndpoint,
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.athenaClient, nil
}

func (cc *CommonConfig) GetProvider(ctx context.Context) (string, error) {
	installationService, err := cc.GetInstallationService(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	installationInfo, err := installationService.GetInfo(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return installationInfo.Provider, nil
}

func (cc *CommonConfig) GetClient() (k8sclient.Interface, error) {
	if cc.k8sClient != nil {
		return cc.k8sClient, nil
	}

	restConfig, err := cc.clientGetter.ToRESTConfig()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	config := k8sclient.ClientsConfig{
		Logger:        cc.logger,
		RestConfig:    restConfig,
		SchemeBuilder: scheme.NewSchemeBuilder(),
	}

	cc.k8sClient, err = k8sclient.NewClients(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.k8sClient, nil
}
