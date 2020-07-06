package login

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	contextPrefix = "gs-"
)

func generateKubeContextName(installationCodeName string) string {
	return fmt.Sprintf("%s%s", contextPrefix, installationCodeName)
}

func isKubeContext(s string) bool {
	return strings.HasPrefix(s, contextPrefix)
}

func getCodeNameFromKubeContext(c string) string {
	return strings.TrimPrefix(c, contextPrefix)
}

func isCodeName(s string) bool {
	codeNameRegExp, _ := regexp.Compile("^[a-z]+$")
	return codeNameRegExp.MatchString(s)
}
