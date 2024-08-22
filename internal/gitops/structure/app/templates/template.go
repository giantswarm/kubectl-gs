package app

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
)

//go:embed appcr.yaml.tmpl
var appcr string

//go:embed kustomization.yaml.tmpl
var kustomization string

//go:embed patch_app_userconfig.yaml.tmpl
var patchUserConfig string

//go:embed configmap.yaml.tmpl
var userValuesCm string

//go:embed secret.yaml.tmpl
var userValuesSecret string

// GetOrganizationDirectoryTemplates returns organization directory layout.
func GetAppDirectoryTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "appcr.yaml", Data: appcr},
		common.Template{Name: "configmap.yaml", Data: userValuesCm},
		common.Template{Name: "secret.enc.yaml", Data: userValuesSecret},
		common.Template{Name: "kustomization.yaml", Data: kustomization},
		common.Template{Name: "patch_app_userconfig.yaml", Data: patchUserConfig},
	}
}
