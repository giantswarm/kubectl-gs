package installation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/pkg/graphql"
)

const (
	requestTimeout = 15 * time.Second
)

type Installation struct {
	K8sApiURL         string
	K8sInternalApiURL string
	AuthURL           string
	Provider          string
	Codename          string
	CACert            string
	SourcePath        string
}

func New(ctx context.Context, fromUrl, customAthenaUrl string) (*Installation, error) {
	_, internalApiPath, athenaUrl, err := GetBaseAndInternalPath(fromUrl)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var gqlClient graphql.Client
	{
		httpClient := http.DefaultClient
		httpClient.Timeout = requestTimeout

		if customAthenaUrl != "" {
			athenaUrl = customAthenaUrl
		}
		config := graphql.ClientImplConfig{
			HttpClient: httpClient,
			Url:        fmt.Sprintf("%s/graphql", athenaUrl),
		}
		gqlClient, err = graphql.NewClient(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	info, err := getInstallationInfo(ctx, gqlClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sInternalAPI := fmt.Sprintf("https://%s", internalApiPath)
	i := &Installation{
		K8sApiURL:         info.Kubernetes.ApiUrl,
		K8sInternalApiURL: k8sInternalAPI,
		AuthURL:           info.Kubernetes.AuthUrl,
		Provider:          info.Identity.Provider,
		Codename:          info.Identity.Codename,
		CACert:            info.Kubernetes.CaCert,
		SourcePath:        fromUrl,
	}

	return i, nil
}
