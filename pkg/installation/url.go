package installation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
)

const (
	k8sApiUrlPrefix = "g8s"
	happaUrlPrefix  = "happa"
	athenaUrlPrefix = "athena"
)

const (
	UrlTypeInvalid = iota
	UrlTypeK8sApi
	UrlTypeHappa
)

var k8sApiURLRegexp = regexp.MustCompile(fmt.Sprintf("([^.]*)%s.*$", k8sApiUrlPrefix))

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
	return k8sApiURLRegexp.MatchString(u)
}

func isHappaUrl(u string) bool {
	return strings.Contains(u, fmt.Sprintf("%s.", happaUrlPrefix))
}

func GetBasePath(u string) (string, error) {
	// Add https scheme if it doesn't exist.
	urlRegexp := regexp.MustCompile("^http(s)?://.*$")
	if matched := urlRegexp.MatchString(u); !matched {
		u = fmt.Sprintf("https://%s", u)
	}

	path, err := url.ParseRequestURI(u)
	if err != nil {
		return "", microerror.Mask(err)
	}

	switch GetUrlType(path.Host) {
	case UrlTypeK8sApi:
		return k8sApiURLRegexp.FindString(path.Host), nil
	case UrlTypeHappa:
		basePath := strings.Replace(path.Host, fmt.Sprintf("%s.", happaUrlPrefix), "", -1)
		return basePath, nil
	default:
		return "", microerror.Mask(unknownUrlTypeError)
	}
}

func getAthenaUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", athenaUrlPrefix, basePath)
}
