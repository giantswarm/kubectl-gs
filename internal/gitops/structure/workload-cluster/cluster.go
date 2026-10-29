package wcluster

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"go.yaml.in/yaml/v3"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
	fluxkusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/flux-kustomization"
	sigskusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	wctmpl "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/workload-cluster/templates"
)

// NewWorkloadCluster creates a new Workload Cluster directory
// structure.
func NewWorkloadCluster(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	if config.ClusterBase != "" && config.Release != "" {
		config.ClusterUserConfig, err = clusterUserConfigWithRelease(config.ClusterUserConfig, config.Release)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

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

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME.yaml
	wcFile := key.ResourcePath(wcsDir, key.FluxKustomizationFileName(config.WorkloadCluster))

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

	// Workload Cluster can be added without a cluster definition at first, if all user wants is
	// to manage apps for example. Later on, when `kgs` is re-run to add a Workload Cluster from
	// a base the WC_NAME.yaml file must be updated with the right environment variables.
	if config.ClusterBase != "" {
		fsModifiers[wcFile] = fluxkusmod.KustomizationModifier{
			PostBuildEnvs: map[string]string{
				"cluster_description": fmt.Sprintf("%s managed by GitOps", config.WorkloadCluster),
				"release":             config.Release,
			},
		}
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerWorkloadCluster,
			Path: wcDir,
		},
	}

	return &creatorConfig, nil
}

// releaseVersionPath is where the cluster chart's values carry the release version.
var releaseVersionPath = []string{"global", "release", "version"}

// clusterUserConfigWithRelease sets `global.release.version` in the cluster
// app values. All other values the user gave are kept, with their comments
// and order. Cluster charts read the release version from there.
func clusterUserConfigWithRelease(userConfig, release string) (string, error) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(userConfig), &doc)
	if err != nil {
		return "", microerror.Maskf(invalidConfigError, "cluster user config is not valid YAML: %s", err)
	}
	// Empty user configuration unmarshals to a zero node.
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return "", microerror.Maskf(invalidConfigError, "cluster user config must be a mapping")
	}

	err = setValue(doc.Content[0], release, releaseVersionPath...)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(&doc)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = encoder.Close()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// setValue sets the string at the mapping path. Missing keys are appended,
// and values on the way that aren't mappings are replaced by one.
func setValue(node *yaml.Node, value string, path ...string) error {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != path[0] {
			continue
		}

		child := node.Content[i+1]
		if len(path) == 1 {
			child.Kind, child.Tag, child.Value, child.Content = yaml.ScalarNode, "!!str", value, nil
			return nil
		}
		// Replacing an alias would silently drop the values of its anchor.
		if child.Kind == yaml.AliasNode {
			return microerror.Maskf(invalidConfigError, "cluster user config uses an alias at %q, set %s there without an alias", path[0], strings.Join(releaseVersionPath, "."))
		}
		if child.Kind != yaml.MappingNode {
			child.Kind, child.Tag, child.Value, child.Content, child.Style = yaml.MappingNode, "!!map", "", nil, 0
		}
		return setValue(child, value, path[1:]...)
	}

	child := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	if len(path) > 1 {
		child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		err := setValue(child, value, path[1:]...)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: path[0]}, child)

	return nil
}
