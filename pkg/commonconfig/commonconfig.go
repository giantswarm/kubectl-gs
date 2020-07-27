package commonconfig

import (
	"fmt"
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	dataClient "github.com/giantswarm/kubectl-gs/pkg/data/client"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	providerRegexpPattern = `.+\.%s\..+`
)

type CommonConfig struct {
	configFlags genericclioptions.RESTClientGetter
}

func New(cf genericclioptions.RESTClientGetter) *CommonConfig {
	cc := &CommonConfig{
		configFlags: cf,
	}

	return cc
}

func (cc *CommonConfig) GetProvider() (string, error) {
	config, err := cc.configFlags.ToRESTConfig()
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
		provider = key.ProviderKVM
	}

	return provider, nil
}

func (cc *CommonConfig) GetClient(logger micrologger.Logger) (*dataClient.Client, error) {
	restConfig, err := cc.configFlags.ToRESTConfig()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	config := dataClient.Config{
		Logger:        logger,
		K8sRestConfig: restConfig,
	}

	client, err := dataClient.New(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}
