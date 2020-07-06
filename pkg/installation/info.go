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
