package app

import (
	"fmt"
	"regexp"
)

const (
	appVersionLocator = `version:\s([v0-9.]*).*\n`
	appVersionUpdater = "version: $1 # {\"$$imagepolicy\": \"default:%s:tag\"}\n"
)

type AppModifier struct {
	ImagePolicyToAdd string
}

// PostExecute modifies App CRs after creating the necessary
// resources first.
func (app AppModifier) Execute(rawYaml []byte) ([]byte, error) {

	if app.ImagePolicyToAdd != "" {
		re := regexp.MustCompile(appVersionLocator)
		rawYaml = re.ReplaceAll(rawYaml, []byte(fmt.Sprintf(appVersionUpdater, app.ImagePolicyToAdd)))
	}

	return rawYaml, nil
}
