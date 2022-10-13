package normalize

import (
	"strings"
	"unicode"
)

const (
	maxDNSLabelLength = 63
)

// AsDNSLabelName normalizes input string to be valid DNS label name so that it
// can be used as Kubernetes object identifier such as namespace name.
//
// NOTE: This function returns an empty string if input string consists of only
// non-allowed characters.
//
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
//
// Source: https://github.com/giantswarm/azure-operator/blob/master/pkg/normalize/normalize.go
func AsDNSLabelName(v string) string {
	var xs []rune

	for _, x := range strings.ToLower(v) {

		// Is x in whitelisted characters?
		if x == '-' || unicode.IsDigit(x) || ('a' <= x && x <= 'z') {
			xs = append(xs, x)
		} else if len(xs) > 0 && xs[len(xs)-1] != '-' {
			// If not, append dash and coalesce consecutive ones.
			xs = append(xs, '-')
		}
	}

	// Ensure that the string doesn't start or end with dash.
	for len(xs) > 0 {
		if xs[0] == '-' {
			xs = xs[1:]
			continue
		}

		if xs[len(xs)-1] == '-' {
			xs = xs[:len(xs)-1]
			continue
		}

		break
	}

	if len(xs) > maxDNSLabelLength {
		xs = xs[:maxDNSLabelLength]
	}

	return string(xs)
}
