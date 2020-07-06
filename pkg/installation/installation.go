package installation

import (
	"github.com/giantswarm/microerror"
)

const (
	apiURLPrefix  = "api"
	authURLPrefix = "dex"
)

type Installation struct {
	ApiURL    string
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
	apiUrl := getGiantSwarmAPIUrl(basePath)
	authUrl := getAuthUrl(basePath)

	installationInfo, err := getInstallationInfo(apiUrl)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	i := &Installation{
		ApiURL:    apiUrl,
		K8sApiURL: fromUrl,
		AuthURL:   authUrl,
		Provider:  installationInfo.Provider,
		Codename:  installationInfo.Name,
		CACert:    installationInfo.K8sCaCert,
	}

	return i, nil
}
