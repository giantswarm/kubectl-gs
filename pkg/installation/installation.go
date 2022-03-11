package installation

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/graphql"
)

const (
	requestTimeout = 15 * time.Second

	// management cluster internal api prefix
	internalAPIPrefix = "internal"
)

type Config struct {
	AthenaClient graphql.Client
}

type Service struct {
	athenaClient graphql.Client
}

type Info struct {
	K8sApiURL         string
	K8sInternalApiURL string
	AuthURL           string
	Provider          string
	Codename          string
	CACert            string
}

func New(config Config) (*Service, error) {
	if config.AthenaClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AthenaClient must not be empty", config)
	}

	return &Service{
		athenaClient: config.AthenaClient,
	}, nil
}

func (s *Service) GetInfo(ctx context.Context) (*Info, error) {
	info, err := getInstallationInfo(ctx, s.athenaClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sInternalAPIURL := info.Kubernetes.ApiUrl
	if info.Identity.Provider == key.ProviderAWS {
		parsedURL, err := url.Parse(info.Kubernetes.ApiUrl)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sInternalAPIURL = fmt.Sprintf("https://%s-%s", internalAPIPrefix, parsedURL.Host)
	}

	return &Info{
		K8sApiURL:         info.Kubernetes.ApiUrl,
		K8sInternalApiURL: k8sInternalAPIURL,
		AuthURL:           info.Kubernetes.AuthUrl,
		Provider:          info.Identity.Provider,
		Codename:          info.Identity.Codename,
		CACert:            info.Kubernetes.CaCert,
	}, nil
}
