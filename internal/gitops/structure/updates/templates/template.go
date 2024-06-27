package autoupdate

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
)

//go:embed imagepolicy.yaml.tmpl
var imagepolicy string

//go:embed imagerepository.yaml.tmpl
var imagerepository string

//go:embed imageupdate.yaml.tmpl
var imageupdate string

// GetAutomaticUpdatesTemplates returns organization directory layout.
func GetAutomaticUpdatesTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "imageupdate.yaml", Data: imageupdate},
	}
}

func GetAppImageUpdatesTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "imagepolicy.yaml", Data: imagepolicy},
		common.Template{Name: "imagerepository.yaml", Data: imagerepository},
	}
}
