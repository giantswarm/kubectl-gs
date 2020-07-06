package installation

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/giantswarm/microerror"
)

const (
	apiUrlPrefix    = "api"
	k8sApiUrlPrefix = "g8s"
	happaUrlPrefix  = "happa"
	authUrlPrefix   = "dex"
)

const (
	UrlTypeInvalid = iota
	UrlTypeK8sApi
	UrlTypeHappa
)

func GetUrlType(u string) int {
	switch {
	case isHappaUrl(u):
		return UrlTypeHappa
	case isK8sApiUrl(u):
		return UrlTypeK8sApi
	default:
		return UrlTypeInvalid
	}
}

func isK8sApiUrl(u string) bool {
	return strings.Contains(u, fmt.Sprintf("%s.", k8sApiUrlPrefix))
}

func isHappaUrl(u string) bool {
	return strings.Contains(u, fmt.Sprintf("%s.", happaUrlPrefix))
}

func getBasePath(u string) (string, error) {
	path, err := url.Parse(u)
	if err != nil {
		return "", microerror.Mask(err)
	}

	switch GetUrlType(path.Host) {
	case UrlTypeK8sApi:
		return path.Host, nil
	case UrlTypeHappa:
		basePath := strings.Replace(path.Host, fmt.Sprintf("%s.", happaUrlPrefix), "", -1)
		return basePath, nil
	default:
		return "", microerror.Mask(unknownUrlTypeError)
	}
}

func getGiantSwarmApiUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", apiUrlPrefix, basePath)
}

func getK8sApiUrl(basePath string) string {
	return fmt.Sprintf("https://%s", basePath)
}

func getAuthUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", authUrlPrefix, basePath)
}
