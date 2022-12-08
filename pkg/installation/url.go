package installation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
)

const (
	wcK8sApiUrlPrefix = "k8s"
	k8sApiUrlPrefix   = "g8s"
	happaUrlPrefix    = "happa"
	athenaUrlPrefix   = "athena"
	internalPrefix    = "internal"
)

const (
	UrlTypeInvalid = iota
	UrlTypeK8sApi
	UrlTypeHappa
	UrlTypeWcK8sApi
)

var k8sApiURLRegexp = regexp.MustCompile(fmt.Sprintf("([^.]*)%s.*$", k8sApiUrlPrefix))
var wcK8sApiUrlRegexp = regexp.MustCompile(fmt.Sprintf("^api\\.[0-9a-zA-Z]{5,10}\\.%s.*$", wcK8sApiUrlPrefix))

func GetUrlType(u string) int {
	switch {
	case isHappaUrl(u):
		return UrlTypeHappa
	case isWcK8sApiUrl(u):
		return UrlTypeWcK8sApi
	case isK8sApiUrl(u):
		return UrlTypeK8sApi
	default:
		return UrlTypeInvalid
	}
}

func isK8sApiUrl(u string) bool {
	u = strings.SplitN(u, ":", 2)[0]
	return k8sApiURLRegexp.MatchString(u) || strings.HasPrefix(u, "api.")
}

func isWcK8sApiUrl(u string) bool {
	u = strings.SplitN(u, ":", 2)[0]
	return wcK8sApiUrlRegexp.MatchString(u)
}

func isHappaUrl(u string) bool {
	return strings.Contains(u, fmt.Sprintf("%s.", happaUrlPrefix))
}

func GetBaseAndInternalPath(u string) (string, string, error) {
	// Add https scheme if it doesn't exist.
	urlRegexp := regexp.MustCompile("^http(s)?://.*$")
	if matched := urlRegexp.MatchString(u); !matched {
		u = fmt.Sprintf("https://%s", u)
	}

	path, err := url.ParseRequestURI(u)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	urlType := GetUrlType(path.Host)
	switch urlType {
	case UrlTypeWcK8sApi:
		return getSanitizedApiPath(path.Host), getInternalPath(path.Host), nil
	case UrlTypeK8sApi:
		if p := k8sApiURLRegexp.FindString(path.Host); len(p) > 0 {
			return p, getInternalPath(p), nil
		}
		p := getSanitizedApiPath(path.Host)
		return p, getInternalPath(p), nil
	case UrlTypeHappa:
		basePath := strings.Replace(path.Host, fmt.Sprintf("%s.", happaUrlPrefix), "", -1)
		return basePath, getInternalPath(basePath), nil
	default:
		return "", "", microerror.Mask(unknownUrlTypeError)
	}
}

func getSanitizedApiPath(u string) string {
	p := strings.SplitN(u, ":", 2)[0]
	return strings.SplitN(p, ".", 2)[1]
}

func getInternalPath(u string) string {
	p := strings.SplitN(u, ":", 2)[0]
	return fmt.Sprintf("%s-%s", internalPrefix, p)
}

func getAthenaUrl(basePath string) string {
	return fmt.Sprintf("https://%s.%s", athenaUrlPrefix, basePath)
}
