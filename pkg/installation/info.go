package installation

import (
	"encoding/json"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/pkg/graphql"
)

const infoQuery = `
query GetInfo {
  identity {
    provider
    codename
  }

  kubernetes {
    apiUrl
    authUrl
    caCert
  }
}
`

type installationInfoIdentity struct {
	Provider string `json:"provider"`
	Codename string `json:"codename"`
}

type installationInfoKubernetes struct {
	ApiUrl  string `json:"apiUrl"`
	AuthUrl string `json:"authUrl"`
	CaCert  string `json:"caCert"`
}

type installationInfo struct {
	Identity   installationInfoIdentity   `json:"identity"`
	Kubernetes installationInfoKubernetes `json:"kubernetes"`
}

func getInstallationInfo(gqlClient graphql.Client) (installationInfo, error) {
	result, err := gqlClient.ExecuteQuery(infoQuery, nil)
	if err != nil {
		return installationInfo{}, microerror.Maskf(cannotGetInstallationInfoError, "make sure you're connected to the internet and that the Athena service is up and running\n%s", err.Error())
	}

	var info installationInfo
	err = json.Unmarshal(*result, &info)
	if err != nil {
		return installationInfo{}, microerror.Maskf(cannotGetInstallationInfoError, "the fetched installation information has an invalid format")
	}

	return info, nil
}
