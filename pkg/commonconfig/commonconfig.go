package commonconfig

import (
	"fmt"
	"regexp"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/scheme"
)

const (
	providerRegexpPattern = `.+\.%s\..+`
)

type CommonConfig struct {
	ConfigFlags *genericclioptions.RESTClientGetter
}

func New(cf genericclioptions.RESTClientGetter) *CommonConfig {
	cc := &CommonConfig{
		ConfigFlags: &cf,
	}

	return cc
}

func (cc *CommonConfig) GetConfigFlags() genericclioptions.RESTClientGetter {
	return *cc.ConfigFlags
}

func (cc *CommonConfig) GetProvider() (string, error) {
	config, err := cc.GetConfigFlags().ToRESTConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	awsRegexp := regexp.MustCompile(fmt.Sprintf(providerRegexpPattern, key.ProviderAWS))
	azureRegexp := regexp.MustCompile(fmt.Sprintf(providerRegexpPattern, key.ProviderAzure))

	var provider string
	switch {
	case awsRegexp.MatchString(config.Host):
		provider = key.ProviderAWS

	case azureRegexp.MatchString(config.Host):
		provider = key.ProviderAzure

	default:
		provider = key.ProviderOpenStack
	}

	return provider, nil
}

func (cc *CommonConfig) GetClient(logger micrologger.Logger) (k8sclient.Interface, error) {
	restConfig, err := cc.GetConfigFlags().ToRESTConfig()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	config := k8sclient.ClientsConfig{
		Logger:        logger,
		RestConfig:    restConfig,
		SchemeBuilder: scheme.NewSchemeBuilder(),
	}

	k8sClients, err := k8sclient.NewClients(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return k8sClients, nil
}

func (cc *CommonConfig) GetContextOverride() string {
	if c, ok := cc.GetConfigFlags().(*genericclioptions.ConfigFlags); ok && c.Context != nil && len(*c.Context) > 0 {
		return *c.Context
	}
	return ""
}

func (cc *CommonConfig) GetTokenOverride() string {
	if c, ok := cc.GetConfigFlags().(*genericclioptions.ConfigFlags); ok && c.BearerToken != nil && len(*c.BearerToken) > 0 {
		return *c.BearerToken
	}
	return ""
}

func (cc *CommonConfig) GetNamespace() (string, bool, error) {
	return cc.GetConfigFlags().ToRawKubeConfigLoader().Namespace()
}

func (cc *CommonConfig) GetConfigAccess() clientcmd.ConfigAccess {
	return cc.GetConfigFlags().ToRawKubeConfigLoader().ConfigAccess()
}
