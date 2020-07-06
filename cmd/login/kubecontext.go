package login

import (
	"fmt"
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
