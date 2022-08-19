package structure

import (
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
	sigskusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/workload-cluster/templates"
)

// NewWorkloadCluster creates a new Workload Cluster directory
// structure.
func NewWorkloadCluster(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-cluster/MC_NAME
	mcDir := key.BaseDirPath(config.ManagementCluster, "", "")

	// Holds management-cluster/MC_NAME/secrets
	secretsDir := key.ResourcePath(mcDir, key.SecretsDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, "")

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	// or
	// management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/mapi
	mapiDir := wcDir
	if !config.SkipMAPI {
		mapiDir = key.ResourcePath(wcDir, key.MapiDirName())
	}

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps
	appsDir := key.ResourcePath(mapiDir, key.AppsDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/cluster
	clusterDir := key.ResourcePath(mapiDir, key.ClusterDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/kustomization.yaml
	wcsKusFile := key.ResourcePath(wcsDir, key.SigsKustomizationFileName())

	fsObjects := []*creator.FsObject{}

	// Adding the `MC_NAME/secrets/WC_NAME.gpgkey.enc.yaml. We start here since it's the
	// highest layer in the structure. We add an empty secret there, that serves
	// as the SOPS private keys storage. It is empty by default and can be populated by the
	// `add encryption` command.
	// The need for this here may be questioned, but the rationale is: it is much easier and
	// structured this way. If it is not added here, empty, then it would have to be created
	// when encryption is requested later on. But when that happens, we do not have a way to
	// tell whether something exist or not here in the `structure` package. We can only tell
	// creator to create something or modify it, assuming it exists. So instead of tweaking
	// the creator's logic, I decided to drop an empty Secret and edit it with creator's
	// post modifiers.
	err = common.AppendFromTemplate(&fsObjects, secretsDir, wctmpl.GetSecretsDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We then continue at the `workload-clusters` directory. This should already
	// exist at this point, as a result of Organization creation, but we
	// need to point to this directory anyway in order to drop Kustomization
	// there.
	fsObjects = append(fsObjects, creator.NewFsObject(wcsDir, nil, 0))

	// Add Kustomization CR to the `workload-clusters` directory and other
	// files if needed. Currently only Kustomization CR is considered.
	err = common.AppendFromTemplate(&fsObjects, wcsDir, wctmpl.GetWorkloadClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `WC_NAME` specific directory and then add `apps` and `cluster`
	// directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	fsObjects = append(fsObjects, creator.NewFsObject(wcDir, nil, 0))
	if !config.SkipMAPI {
		fsObjects = append(fsObjects, creator.NewFsObject(mapiDir, nil, 0))
	}
	fsObjects = append(
		fsObjects,
		[]*creator.FsObject{
			creator.NewFsObject(appsDir, nil, 0),
			creator.NewFsObject(clusterDir, nil, 0),
		}...,
	)

	// The `apps/*` directory pre-configuration including kustomization.yaml and
	// kubeconfig patch.
	err = common.AppendFromTemplate(&fsObjects, appsDir, wctmpl.GetAppsDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// The `cluster/*` files, aka cluster definition, including `kustomization.yaml`,
	// patches, etc.
	err = common.AppendFromTemplate(&fsObjects, clusterDir, wctmpl.GetClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// After creating all the files and directories, we need creator to run
	// post modifiers, so that cluster is included into `workload-clusters/kustomization.yaml`
	// file.
	fsModifiers := map[string]modifier.Modifier{
		wcsKusFile: sigskusmod.KustomizationModifier{
			ResourcesToAdd: []string{
				fmt.Sprintf("%s.yaml", config.WorkloadCluster),
			},
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
	}

	return &creatorConfig, nil
}
