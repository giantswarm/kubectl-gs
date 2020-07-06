package installation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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

type installationResponse struct {
	Installation installationInfo `json:"installation"`
}

type installationInfo struct {
	Provider  string `json:"provider"`
	K8sCaCert string `json:"k8s_cacert"`
	Name      string `json:"name"`
}

func getInstallationInfo(apiUrl string) (installationInfo, error) {
	res, err := http.Get(apiUrl) // #nosec G107
	if err != nil {
		return installationInfo{}, microerror.Mask(err)
	}

	defer res.Body.Close()

	result := installationResponse{}
	{
		err = json.NewDecoder(res.Body).Decode(&result)
		if err != nil {
			return installationInfo{}, microerror.Maskf(cannotGetInstallationInfo, "make sure you're connected to the internet and behind a VPN")
		}

		result.Installation.K8sCaCert = parseCertificate(result.Installation.K8sCaCert)
	}

	return result.Installation, nil
}

func getBasePath(k8sAPIUrl string) (string, error) {
	k8sURL, err := url.Parse(k8sAPIUrl)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return k8sURL.Host, nil
}

func getGiantSwarmAPIUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", apiURLPrefix, basePath)
}

func getAuthUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", authURLPrefix, basePath)
}
