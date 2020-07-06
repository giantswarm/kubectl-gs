package installation

import (
	"fmt"
	"net/url"

	"github.com/giantswarm/microerror"
)

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
