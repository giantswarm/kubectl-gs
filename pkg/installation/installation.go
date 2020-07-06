package installation

import (
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
	apiUrl := getGiantSwarmApiUrl(basePath)
	authUrl := getAuthUrl(basePath)

	installationInfo, err := getInstallationInfo(apiUrl)
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
