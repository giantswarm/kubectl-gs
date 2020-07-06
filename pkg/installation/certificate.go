package installation

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	yamlMultiLinePrefix = 10

	certPrefix = "-----BEGIN CERTIFICATE-----"
	certSuffix = "-----END CERTIFICATE-----"
)

var (
	certRegexp = regexp.MustCompile(fmt.Sprintf(`%s (.*) %s`, certPrefix, certSuffix))
)

func parseCertificate(cert string) string {
	var certWithoutPrefixAndSuffix string
	{
		// Remove yaml multi-line prefix.
		if len(cert) < (yamlMultiLinePrefix) {
			return ""
		}
		cert = cert[yamlMultiLinePrefix:]

		submatch := certRegexp.FindAllStringSubmatch(cert, -1)
		if len(submatch) == 0 {
			return ""
		}
		certWithoutPrefixAndSuffix = submatch[0][1]
	}

	// Replace spaces with newlines.
	cert = strings.Replace(certWithoutPrefixAndSuffix, " ", "\n", -1)

	// Add back suffix and prefix.
	cert = fmt.Sprintf("%s\n%s\n%s", certPrefix, cert, certSuffix)

	return cert
}
