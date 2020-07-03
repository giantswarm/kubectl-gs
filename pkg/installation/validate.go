package installation

import (
	"net/url"
	"strings"
)

const (
	k8sAPIUrlPrefix = "g8s"
)

func ValidateK8sAPIUrl(u string) bool {
	k8sURL, err := url.Parse(u)
	if err != nil {
		return false
	}

	if k8sURL.Scheme != "https" {
		return false
	}

	if !strings.HasPrefix(k8sURL.Host, k8sAPIUrlPrefix) {
		return false
	}

	return true
}
