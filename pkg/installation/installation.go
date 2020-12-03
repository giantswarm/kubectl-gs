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

	var installationInfo installationInfo
	{
		client := http.DefaultClient
		for _, apiUrl := range apiUrls {
			installationInfo, err = getInstallationInfo(client, apiUrl)
			if IsCannotGetInstallationInfo(err) {
				continue
			} else if err != nil {
				return nil, microerror.Mask(err)
			}

			break
		}

		// None of the urls was correct. Let's throw an error.
		if err != nil {
			return nil, microerror.Mask(err)
		}
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
