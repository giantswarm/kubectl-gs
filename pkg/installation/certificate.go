package installation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
)

const (
	yamlMultiLinePrefix = 10

	certPrefix = "-----BEGIN CERTIFICATE-----"
	certSuffix = "-----END CERTIFICATE-----"
)

var (
	certRegexp = regexp.MustCompile(fmt.Sprintf(`%s (.*) %s`, certPrefix, certSuffix))
)

func parseCertificate(cert string) (string, error) {
	var certWithoutPrefixAndSuffix string
	{
		// Remove yaml multi-line prefix.
		if len(cert) < (yamlMultiLinePrefix) {
			return "", microerror.Mask(cannotParseCertificateError)
		}
		cert = cert[yamlMultiLinePrefix:]

		submatch := certRegexp.FindAllStringSubmatch(cert, -1)
		if len(submatch) == 0 {
			return "", microerror.Mask(cannotParseCertificateError)
		}
		certWithoutPrefixAndSuffix = submatch[0][1]
	}

	// Replace spaces with newlines.
	cert = strings.Replace(certWithoutPrefixAndSuffix, " ", "\n", -1)

	// Add back suffix and prefix.
	cert = fmt.Sprintf("%s\n%s\n%s", certPrefix, cert, certSuffix)

	return cert, nil
}
