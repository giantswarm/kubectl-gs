package installation

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	k8sApiUrlPrefix = "g8s"
	happaUrlPrefix  = "happa"
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
	u = strings.SplitN(u, ":", 2)[0]
	return k8sApiURLRegexp.MatchString(u) || (strings.HasPrefix(u, "api.") && strings.HasSuffix(u, ".gigantic.io"))
}

func isHappaUrl(u string) bool {
	return strings.Contains(u, fmt.Sprintf("%s.", happaUrlPrefix))
}
