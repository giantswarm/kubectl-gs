package organization

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	orgtmpl "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/organization/templates"
)

// NewOrganization creates a new Organization directory
// structure.
func NewOrganization(config common.StructureConfig) (*creator.CreatorConfig, error) {
	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/namespace
	namespaceDir := key.ResourcePath(orgDir, key.NamespaceDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR definition, the organisation's namespace
	// Kustomization in `namespace.yaml`, and the `kustomization.yaml` that
	// lists them for the management cluster's Kustomization. Then create the
	// `namespace` directory, which `namespace.yaml` applies into the
	// organisation's namespace, and the `workload-clusters` directory with an
	// empty `kustomization.yaml`, which `namespace/kustomization.yaml` lists.
	fsObjects, err := directories([]directory{
		{path: orgDir, templates: orgtmpl.GetOrganizationDirectoryTemplates},
		{path: namespaceDir, templates: orgtmpl.GetNamespaceDirectoryTemplates},
		{path: wcsDir, templates: orgtmpl.GetWorkloadClustersDirectoryTemplates},
	}, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerOrganization,
			Path: orgDir,
		},
	}

	return &creatorConfig, nil
}

// directory is a directory of the organisation and the files it holds.
type directory struct {
	path      string
	templates func() []common.Template
}

// directories returns the directories and the files they hold, in order.
func directories(dirs []directory, config common.StructureConfig) ([]*creator.FsObject, error) {
	fsObjects := []*creator.FsObject{}
	for _, d := range dirs {
		fsObjects = append(fsObjects, creator.NewFsObject(d.path, nil, 0))

		err := common.AppendFromTemplate(&fsObjects, d.path, d.templates, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return fsObjects, nil
}
