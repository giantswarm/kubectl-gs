package installation

import (
	"fmt"
	"net/http"
	"time"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/pkg/graphql"
)

const (
	requestTimeout = 15 * time.Second
)

type Installation struct {
	K8sApiURL string
	AuthURL   string
	Provider  string
	Codename  string
	CACert    string
}

func New(fromUrl string) (*Installation, error) {
	basePath, err := getBasePath(fromUrl)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var gqlClient graphql.Client
	{
		httpClient := http.DefaultClient
		httpClient.Timeout = requestTimeout

		athenaUrl := getAthenaUrl(basePath)
		config := graphql.ClientImplConfig{
			HttpClient: httpClient,
			Url:        fmt.Sprintf("%s/graphql", athenaUrl),
		}
		gqlClient, err = graphql.NewClient(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	info, err := getInstallationInfo(gqlClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	i := &Installation{
		K8sApiURL: info.Kubernetes.ApiUrl,
		AuthURL:   info.Kubernetes.AuthUrl,
		Provider:  info.Identity.Provider,
		Codename:  info.Identity.Codename,
		CACert:    info.Kubernetes.CaCert,
	}

	return i, nil
}
