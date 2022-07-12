package kubeconfig

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ContextPrefix    = "gs-"
	ClientCertSuffix = "-clientcert"
)

type ContextType int

const (
	ContextTypeNone ContextType = iota
	ContextTypeMC
	ContextTypeWC
)

var contextRegexp = regexp.MustCompile(fmt.Sprintf(`^%s([a-z0-9]+)(-[a-z0-9-]+)?$`, ContextPrefix))

// GenerateKubeContextName creates a context name,
// from an installation's code name.
func GenerateKubeContextName(installationCodeName string) string {
	return fmt.Sprintf("%s%s", ContextPrefix, installationCodeName)
}

func GenerateWCKubeContextName(mcKubeContextName string, wcName string) string {
	return fmt.Sprintf("%s-%s", mcKubeContextName, wcName)
}

func GenerateWCClientCertKubeContextName(mcKubeContextName string, wcName string) string {
	return fmt.Sprintf("%s-%s%s", mcKubeContextName, wcName, ClientCertSuffix)
}

// IsKubeContext checks whether the name provided,
// matches our pattern for naming kubernetes contexts.
func IsKubeContext(s string) (bool, ContextType) {
	contextType := GetKubeContextType(s)

	return contextType != ContextTypeNone, contextType
}

// GetClientCertContextName returns the name of the client cert context name for an identifier
func GetClientCertContextName(identifier string) string {
	if !strings.HasPrefix(identifier, ContextPrefix) {
		identifier = ContextPrefix + identifier
	}
	if !strings.HasSuffix(identifier, ClientCertSuffix) {
		identifier = identifier + ClientCertSuffix
	}
	return identifier
}

// GetCodeNameFromKubeContext gets an installation's
// code name, by knowing the context used to reference it.
func GetCodeNameFromKubeContext(c string) string {
	parts := getKubeContextParts(c)
	if len(parts) < 1 {
		return c
	}

	return parts[0]
}

func GetClusterNameFromKubeContext(c string) string {
	parts := getKubeContextParts(c)
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}

// IsCodeName checks whether a provided name is
// an installation's code name.
func IsCodeName(s string) bool {
	codeNameRegExp, _ := regexp.Compile("^[a-z]+$")
	return codeNameRegExp.MatchString(s)
}

// IsWCCodeName checks whether a provided name is
// a WC id with installation code name prefix.
func IsWCCodeName(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return false
	}
	if !IsCodeName(parts[0]) {
		return false
	}
	return true
}

func GetKubeContextType(s string) ContextType {
	contextParts := getKubeContextParts(s)

	switch len(contextParts) {
	case 2:
		return ContextTypeWC
	case 1:
		return ContextTypeMC
	default:
		return ContextTypeNone
	}
}

func getKubeContextParts(s string) []string {
	submatches := contextRegexp.FindStringSubmatch(s)
	if len(submatches) < 1 {
		return nil
	}

	var parts []string
	for _, match := range submatches[1:] {
		if len(match) > 0 {
			parts = append(parts, strings.TrimPrefix(match, "-"))
		}
	}

	return parts
}
