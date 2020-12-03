package installation

import (
	"net/http"

	"github.com/giantswarm/microerror"
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

	k8sApiUrl := getK8sApiUrl(basePath)
	apiUrls := getGiantSwarmApiUrls(basePath)
	authUrl := getAuthUrl(basePath)

	client := http.DefaultClient
	installationInfo, err := getInstallationInfo(client, apiUrls[0])
	if err != nil {
		return nil, microerror.Mask(err)
	}

	i := &Installation{
		K8sApiURL: k8sApiUrl,
		AuthURL:   authUrl,
		Provider:  installationInfo.Provider,
		Codename:  installationInfo.Name,
		CACert:    installationInfo.K8sCaCert,
	}

	return i, nil
}
