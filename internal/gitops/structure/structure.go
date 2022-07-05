package structure

import (
	"bytes"
	"text/template"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	mctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/management-cluster"
	orgtmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/organization"
)

func NewManagementCluster(config McConfig) (*filesystem.Dir, error) {
	var err error

	mcDir := filesystem.NewDir(config.Name)
	for _, t := range mctmpl.GetManagementClusterTemplates() {
		te := template.Must(template.New("files").Parse(t))

		var buf bytes.Buffer
		err = te.Execute(&buf, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		mcDir.AddFile(
			key.FileName(config.Name),
			buf.Bytes(),
		)
	}
	mcDir.AddDirectory(filesystem.NewDir(key.DirectorySecrets))
	mcDir.AddDirectory(filesystem.NewDir(key.DirectorySOPSPublicKeys))
	mcDir.AddDirectory(filesystem.NewDir(key.DirectoryOrganizations))

	return mcDir, nil
}

func NewOrganization(config OrgConfig) (*filesystem.Dir, error) {
	var err error

	orgDir := filesystem.NewDir(config.Name)
	for _, t := range orgtmpl.OrganizationDirectoryTemplates() {
		te := template.Must(template.New("files").Parse(t))

		var buf bytes.Buffer
		err = te.Execute(&buf, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		orgDir.AddFile(
			key.FileName(config.Name),
			buf.Bytes(),
		)
	}
	orgDir.AddDirectory(filesystem.NewDir(key.DirectoryWorkloadClusters))

	return orgDir, nil
}
