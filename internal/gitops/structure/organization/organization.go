package organization

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
	orgtmpl "github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/organization/templates"
)

// NewOrganization creates a new Organization directory
// structure.
func NewOrganization(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR definition.
	fsObjects := []*creator.FsObject{creator.NewFsObject(orgDir, nil, 0)}

	err = common.AppendFromTemplate(&fsObjects, orgDir, orgtmpl.GetOrganizationDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `workload-cluster` directory and populate it with an
	// empty `kustomization.yaml`.
	fsObjects = append(fsObjects, creator.NewFsObject(wcsDir, nil, 0))
	err = common.AppendFromTemplate(&fsObjects, wcsDir, orgtmpl.GetWorkloadClustersDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}
