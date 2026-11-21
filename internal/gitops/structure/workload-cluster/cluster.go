package wcluster

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
	fluxkusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/flux-kustomization"
	sigskusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	wctmpl "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/workload-cluster/templates"
)

const missingNamespace = "%s does not exist. The workload clusters' Kustomizations are applied into the organisation's namespace by the Kustomization it holds, which `kubectl gs gitops add organization` writes. An organisation created with an older kubectl-gs needs it added first, see " + namespaceDocs

const namespaceDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md#move-the-workload-cluster-kustomizations"

// templateConfig is what the workload cluster templates are rendered from.
type templateConfig struct {
	common.StructureConfig

	// AppsKustomization splits the cluster's directory between two
	// Kustomizations: one applying the cluster in `cluster/`, and one applying
	// the rest once the cluster is ready.
	AppsKustomization bool
}

// NewWorkloadCluster creates a new Workload Cluster directory
// structure.
//
// A cluster with a definition, from a base, gets two Flux Kustomizations:
// `WC_NAME.yaml` applies `WC_NAME/cluster`, and `WC_NAME-apps.yaml` applies
// the rest of `WC_NAME` once the cluster is ready. A cluster without one gets
// a single Kustomization applying `WC_NAME`. Either way `WC_NAME/kustomization.yaml`
// lists what they apply, and never `out-of-band`, which goes straight into the
// workload cluster.
func NewWorkloadCluster(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	tc := templateConfig{
		StructureConfig:   config,
		AppsKustomization: config.ClusterBase != "" && !config.KeepSingleKustomization,
	}

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, "")

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/cluster
	clusterDir := key.ResourcePath(wcDir, key.ClusterDirName())

	fsObjects := []*creator.FsObject{}

	// We start at the `workload-clusters` directory. This should already
	// exist at this point, as a result of Organization creation, but we
	// need to point to this directory anyway in order to drop the
	// Kustomizations there.
	fsObjects = append(fsObjects, creator.NewFsObject(wcsDir, nil, 0))

	err = common.AppendFromTemplate(&fsObjects, wcsDir, wctmpl.GetWorkloadClustersDirectoryTemplates, tc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create the `WC_NAME` directory, with the `kustomization.yaml` that
	// lists what the Kustomizations apply, and then add the `apps` and
	// `cluster` directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	fsObjects = append(fsObjects, creator.NewFsObject(wcDir, nil, 0))

	err = common.AppendFromTemplate(&fsObjects, wcDir, wctmpl.GetWorkloadClusterDirectoryTemplates, tc)
	if err != nil {
		return nil, microerror.Mask(err)
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
	err = common.AppendFromTemplate(&fsObjects, appsDir, wctmpl.GetAppsDirectoryTemplates, tc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// The `cluster/*` files, aka cluster definition, including `kustomization.yaml`,
	// patches, etc.
	err = common.AppendFromTemplate(&fsObjects, clusterDir, wctmpl.GetClusterDirectoryTemplates, tc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	refuseMapiPath, refuseMapi := common.RefuseMapiCluster(config)

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: postModifiers(tc, wcsDir, wcDir),
		PreValidators: map[string]func(*afero.Afero, string) error{
			key.ResourcePath(orgDir, key.NamespaceFileName()): requireNamespaceKustomization,
		},
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerWorkloadCluster,
			Path: wcDir,
		},
		PreValidators: map[string]func(*afero.Afero, string) error{
			refuseMapiPath: refuseMapi,
		},
	}

	return &creatorConfig, nil
}

// requireNamespaceKustomization refuses a cluster in an organisation without
// its namespace Kustomization, which applies the cluster's Kustomizations into
// the organisation's namespace, and which they depend on.
func requireNamespaceKustomization(fs *afero.Afero, path string) error {
	ok, err := fs.Exists(path)
	if err != nil {
		return microerror.Mask(err)
	}
	if ok {
		return nil
	}

	return microerror.Maskf(creator.ValidationError, missingNamespace, path)
}

// postModifiers returns the changes to files that may exist already: the
// cluster's Kustomizations go into `workload-clusters/kustomization.yaml`, and
// a base added to a cluster with a single Kustomization goes into what it
// applies.
func postModifiers(tc templateConfig, wcsDir, wcDir string) map[string]modifier.Modifier {
	kustomizations := []string{key.FluxKustomizationFileName(tc.WorkloadCluster)}
	if tc.AppsKustomization {
		kustomizations = append(kustomizations, key.FluxKustomizationFileName(tc.WorkloadCluster+"-apps"))
	}

	fsModifiers := map[string]modifier.Modifier{
		key.ResourcePath(wcsDir, key.SigsKustomizationFileName()): sigskusmod.KustomizationModifier{
			ResourcesToAdd: kustomizations,
		},
	}

	if tc.ClusterBase == "" {
		return fsModifiers
	}

	// Workload Cluster can be added without a cluster definition at first, if all user wants is
	// to manage apps for example. Later on, when `kgs` is re-run to add a Workload Cluster from
	// a base the WC_NAME.yaml file must be updated with the right environment variables, and
	// its single Kustomization must apply the `cluster` directory too.
	fsModifiers[key.ResourcePath(wcsDir, key.FluxKustomizationFileName(tc.WorkloadCluster))] = fluxkusmod.KustomizationModifier{
		PostBuildEnvs: map[string]string{
			"cluster_description": fmt.Sprintf("%s managed by GitOps", tc.WorkloadCluster),
			"release":             tc.Release,
		},
	}

	if !tc.AppsKustomization {
		fsModifiers[key.ResourcePath(wcDir, key.SigsKustomizationFileName())] = sigskusmod.KustomizationModifier{
			ResourcesToAdd: []string{key.ClusterDirName() + "/"},
		}
	}

	return fsModifiers
}
