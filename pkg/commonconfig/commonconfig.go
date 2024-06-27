package commonconfig

import (
	"context"
	"fmt"
	"regexp"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/v4/internal/key"
	"github.com/giantswarm/kubectl-gs/v4/pkg/installation"
	"github.com/giantswarm/kubectl-gs/v4/pkg/scheme"
)

const (
	providerRegexpPattern = `.+\.%s\..+`
)

type CommonConfig struct {
	ConfigFlags  *genericclioptions.RESTClientGetter
	installation *installation.Installation
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

func (cc *CommonConfig) GetInstallation(ctx context.Context, path, athenaUrl string) (*installation.Installation, error) {
	if path == "" {
		config, err := cc.GetConfigFlags().ToRESTConfig()
		if err != nil {
			return nil, microerror.Mask(err)
		}
		path = config.Host
	}
	if cc.installation == nil || cc.installation.SourcePath != path {
		i, err := installation.New(ctx, path, athenaUrl)
		cc.installation = i
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	return cc.installation, nil
}

func (cc *CommonConfig) GetProviderFromConfig(ctx context.Context, athenaUrl string) (string, error) {
	config, err := cc.GetConfigFlags().ToRESTConfig()
	if err != nil {
		return "", err
	}

	i, err := cc.GetInstallation(ctx, config.Host, athenaUrl)
	if err == nil {
		return i.Provider, nil
	}

	awsRegexp := regexp.MustCompile(fmt.Sprintf(providerRegexpPattern, key.ProviderAWS))
	azureRegexp := regexp.MustCompile(fmt.Sprintf(providerRegexpPattern, key.ProviderAzure))
	capaRegexp := regexp.MustCompile(fmt.Sprintf(providerRegexpPattern, key.ProviderCAPA))

	var provider string
	switch {
	case awsRegexp.MatchString(config.Host):
		provider = key.ProviderAWS
	case azureRegexp.MatchString(config.Host):
		provider = key.ProviderAzure
	case capaRegexp.MatchString(config.Host):
		provider = key.ProviderCAPA
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
