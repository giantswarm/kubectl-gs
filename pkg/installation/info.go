package installation

import (
	"encoding/json"
	"net/http"

	"github.com/giantswarm/microerror"
)

type installationResponse struct {
	Installation installationInfo `json:"installation"`
}

type installationInfo struct {
	Provider  string `json:"provider"`
	K8sCaCert string `json:"k8s_cacert"`
	Name      string `json:"name"`
}

func getInstallationInfo(httpClient *http.Client, apiUrl string) (installationInfo, error) {
	res, err := httpClient.Get(apiUrl) // #nosec G107
	if err != nil {
		return installationInfo{}, microerror.Maskf(cannotGetInstallationInfo, "Make sure you're connected to the internet and that the Giant Swarm API is up and running\n%v", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return installationInfo{}, microerror.Maskf(cannotGetInstallationInfo, "Make sure you're behind the correct VPN")
	}

	defer res.Body.Close()

	result := installationResponse{}
	{
		err = json.NewDecoder(res.Body).Decode(&result)
		if err != nil {
			return installationInfo{}, microerror.Maskf(cannotGetInstallationInfo, "API response has invalid format")
		}

		result.Installation.K8sCaCert, err = parseCertificate(result.Installation.K8sCaCert)
		if err != nil {
			return installationInfo{}, microerror.Mask(err)
		}
	}

	return result.Installation, nil
}
