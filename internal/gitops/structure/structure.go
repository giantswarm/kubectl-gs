package structure

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	mctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/management-cluster"
	orgtmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/organization"
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/workload-cluster"
)

func NewManagementCluster(config McConfig) (*filesystem.Dir, error) {
	var err error

	// Create `MC_NAME` directory and add bunch of other
	// directories there for SOPS keys, organizations, and secrets.
	mcDir := filesystem.NewDir(config.Name)
	err = addFilesFromTemplate(mcDir, mctmpl.GetManagementClusterTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	mcDir.AddDirectory(filesystem.NewDir(key.DirectorySecrets))
	mcDir.AddDirectory(filesystem.NewDir(key.DirectorySOPSPublicKeys))
	mcDir.AddDirectory(filesystem.NewDir(key.DirectoryOrganizations))

	return mcDir, nil
}

func NewOrganization(config OrgConfig) (*filesystem.Dir, error) {
	var err error

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR
	orgDir := filesystem.NewDir(config.Name)
	err = addFilesFromTemplate(orgDir, orgtmpl.GetOrganizationDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `workload-cluster` directory and populate it with
	// empty `kustomization.yaml`.
	wcDir := filesystem.NewDir(key.DirectoryWorkloadClusters)
	err = addFilesFromTemplate(wcDir, orgtmpl.GetWorkloadClustersDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	orgDir.AddDirectory(wcDir)

	return orgDir, nil
}

func NewWorkloadCluster(config WcConfig) (*filesystem.Dir, error) {
	var err error

	// Create Dir pointing to the `workload-clusters` directory. This should
	// already exist at this point, as a result of Organization creation, but
	// we need to point to this directory anyway in order to drop Kustomization
	// there.
	wcsDir := filesystem.NewDir(key.DirectoryWorkloadClusters)
	err = addFilesFromTemplate(wcsDir, wctmpl.GetWorkloadClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `WC_NAME` specific directory and then add `apps` and `cluster`
	// directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	wcDir := filesystem.NewDir(config.Name)
	wcsDir.AddDirectory(wcDir)

	// The `apps` directory
	wcDir.AddDirectory(filesystem.NewDir(key.DirectoryClusterApps))

	// The `cluster` directory
	wcDefDir := filesystem.NewDir(key.DirectoryClusterDefinition)

	// The `cluster/*` files, aka cluster definition
	err = addFilesFromTemplate(wcDefDir, wctmpl.GetClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	wcDir.AddDirectory(wcDefDir)

	return wcsDir, nil
}

func addFilesFromTemplate(dir *filesystem.Dir, templates func() map[string]string, config interface{}) error {
	var err error

	for n, t := range templates() {
		nameTemplate := template.Must(template.New("name").Parse(n))
		var name bytes.Buffer
		err = nameTemplate.Execute(&name, config)
		if err != nil {
			return microerror.Mask(err)
		}
		contentTemplate := template.Must(template.New("files").Funcs(sprig.TxtFuncMap()).Parse(t))

		var content bytes.Buffer
		err = contentTemplate.Execute(&content, config)
		if err != nil {
			return microerror.Mask(err)
		}

		dir.AddFile(
			key.FileName(name.String()),
			content.Bytes(),
		)
	}

	return nil
}
