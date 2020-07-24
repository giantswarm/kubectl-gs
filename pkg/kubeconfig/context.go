package kubeconfig

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ContextPrefix = "gs-"
)

// GenerateKubeContextName creates a context name,
// from an installation's code name.
func GenerateKubeContextName(installationCodeName string) string {
	return fmt.Sprintf("%s%s", ContextPrefix, installationCodeName)
}

// IsKubeContext checks whether the name provided,
// matches our pattern for naming kubernetes contexts.
func IsKubeContext(s string) bool {
	return strings.HasPrefix(s, ContextPrefix)
}

// GetCodeNameFromKubeContext gets an installation's
// code name, by knowing the context used to reference it.
func GetCodeNameFromKubeContext(c string) string {
	return strings.TrimPrefix(c, ContextPrefix)
}

// IsCodeName checks whether a provided name is
// an installation's code name.
func IsCodeName(s string) bool {
	codeNameRegExp, _ := regexp.Compile("^[a-z]+$")
	return codeNameRegExp.MatchString(s)
}
