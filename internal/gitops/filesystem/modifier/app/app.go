package app

import (
	"fmt"
	"regexp"
)

const (
	appVersionLocator = `version:\s([v0-9.]*).*\n`
	appVersionUpdater = "version: $1 # {\"$$imagepolicy\": \"%s:%s:tag\"}\n"
)

type AppModifier struct {
	ImagePolicyToAdd map[string]string
}

// PostExecute modifies App CRs after creating the necessary
// resources first.
func (app AppModifier) Execute(rawYaml []byte) ([]byte, error) {

	for n, p := range app.ImagePolicyToAdd {
		re := regexp.MustCompile(appVersionLocator)
		rawYaml = re.ReplaceAll(rawYaml, []byte(fmt.Sprintf(appVersionUpdater, n, p)))
	}

	return rawYaml, nil
}
