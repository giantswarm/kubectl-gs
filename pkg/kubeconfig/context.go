package kubeconfig

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ContextPrefix = "gs-"
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

// IsKubeContext checks whether the name provided,
// matches our pattern for naming kubernetes contexts.
func IsKubeContext(s string) (bool, ContextType) {
	contextType := GetKubeContextType(s)

	return contextType != ContextTypeNone, contextType
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
