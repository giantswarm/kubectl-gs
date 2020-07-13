package login

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	contextPrefix = "gs-"
)

// generateKubeContextName creates a context name,
// from an installation's code name.
func generateKubeContextName(installationCodeName string) string {
	return fmt.Sprintf("%s%s", contextPrefix, installationCodeName)
}

// isKubeContext checks whether the name provided,
// matches our pattern for naming kubernetes contexts.
func isKubeContext(s string) bool {
	return strings.HasPrefix(s, contextPrefix)
}

// getCodeNameFromKubeContext gets an installation's
// code name, by knowing the context used to reference it.
func getCodeNameFromKubeContext(c string) string {
	return strings.TrimPrefix(c, contextPrefix)
}

// isCodeName checks whether a provided name is
// an installation's code name.
func isCodeName(s string) bool {
	codeNameRegExp, _ := regexp.Compile("^[a-z]+$")
	return codeNameRegExp.MatchString(s)
}
