package mgmtcluster

import (
	_ "embed"

	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/common"
)

//go:embed management-cluster.yaml.tmpl
var managementCluster string

//go:embed private-key.yaml.tmpl
var privateKey string

//go:embed public-key.yaml.tmpl
var publicKey string

// GetManagementClusterTemplates merges .tmpl files for management cluster layer.
func GetManagementClusterTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .ManagementCluster }}.yaml", Data: managementCluster},
	}
}

func GetManagementClusterSecretsTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: "{{ .ManagementCluster }}.gpgkey.enc.yaml", Data: privateKey},
	}
}

func GetManagementClusterSOPSTemplates() []common.Template {
	return []common.Template{
		common.Template{Name: ".sops.master.{{ .EncryptionKeyPair.Fingerprint }}.asc", Data: publicKey},
	}
}
