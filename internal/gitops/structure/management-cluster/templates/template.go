package mgmtcluster

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

//go:embed private-key.yaml.tmpl
var privateKey string

//go:embed public-key.yaml.tmpl
var publicKey string

//go:embed secrets_kustomization.yaml.tmpl
var secretsKustomization string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetManagementClusterTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .ManagementCluster }}.yaml", Data: managementCluster},
	}
}

func GetManagementClusterSecretsTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .ManagementCluster }}.gpgkey.enc.yaml", Data: privateKey},
		common.Template{Name: "kustomization.yaml", Data: secretsKustomization},
	}
}

func GetManagementClusterSOPSTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "master.{{ .EncryptionKeyPair.Fingerprint }}.asc", Data: publicKey},
	}
}
