package migration

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v3"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	clusterUserConfigPatchFile = "patch_cluster_userconfig.yaml"
	clusterConfigPatchFile     = "patch_cluster_config.yaml"
)

// helmReleases moves the layers from App CRs to Flux HelmReleases.
//
// It does not convert App CRs: the HelmRelease has to take over the Helm
// release the App CR installed in the cluster, which can't be done from the
// repository. Layers still holding an App CR fail, and the user moves them
// first. What it does is rewrite what kubectl-gs generated around the App CRs
// once they are gone.
var helmReleases = Migration{
	From:        1,
	Description: "Clusters and apps are deployed with Flux HelmReleases instead of App CRs.",
	Kinds:       []string{metadata.LayerClusterBase, metadata.LayerWorkloadCluster, metadata.LayerApp},
	Apply:       applyHelmReleases,
}

func applyHelmReleases(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
	dir := filepath.Join(repoPath, layer.Path)

	switch layer.Kind {
	case metadata.LayerClusterBase:
		return nil, refuseAppCRs(fs, repoPath, dir, "Move the cluster to a release-<provider> HelmRelease first, see "+appcr.MigrationDocs)
	case metadata.LayerApp:
		return nil, refuseAppCRs(fs, repoPath, dir, "Move the app to a HelmRelease first, see "+appcr.MigrationDocs)
	case metadata.LayerWorkloadCluster:
		return migrateWorkloadCluster(fs, repoPath, dir)
	}

	return nil, nil
}

func refuseAppCRs(fs *afero.Afero, repoPath, dir, hint string) error {
	files, err := appcr.Files(fs, dir)
	if err != nil {
		return microerror.Mask(err)
	}
	if len(files) == 0 {
		return nil
	}

	for i, f := range files {
		files[i] = rel(repoPath, f)
	}

	return microerror.Maskf(appCRError, "%s still deploys App CRs:\n  %s\n%s", rel(repoPath, dir), strings.Join(files, "\n  "), hint)
}

func migrateWorkloadCluster(fs *afero.Afero, repoPath, wcDir string) ([]string, error) {
	// Clusters added without `--skip-mapi` keep their manifests in `mapi`.
	dir := filepath.Join(wcDir, gitopskey.MapiDirName())
	exists, err := fs.DirExists(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !exists {
		dir = wcDir
	}

	var changes []string

	clusterDir := filepath.Join(dir, gitopskey.ClusterDirName())
	exists, err = fs.DirExists(clusterDir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if exists {
		err = refuseAppCRs(fs, repoPath, clusterDir, "Move the cluster to a release-<provider> HelmRelease first, see "+appcr.MigrationDocs)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c, err := migrateClusterUserConfigPatch(fs, repoPath, clusterDir)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	appsDir := filepath.Join(dir, gitopskey.AppsDirName())
	exists, err = fs.DirExists(appsDir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if exists {
		err = refuseAppCRs(fs, repoPath, appsDir, "Move the apps to HelmReleases first, see "+appcr.MigrationDocs)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c, err := removeClusterConfigPatch(fs, repoPath, appsDir)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	return changes, nil
}

// migrateClusterUserConfigPatch turns the patch adding the user config to the
// cluster App into one adding it to the HelmRelease's valuesFrom. A merge
// patch would replace the base's valuesFrom list, so it becomes a JSON 6902
// patch, which needs a target.
func migrateClusterUserConfigPatch(fs *afero.Afero, repoPath, clusterDir string) ([]string, error) {
	patchPath := filepath.Join(clusterDir, clusterUserConfigPatchFile)
	data, err := fs.ReadFile(patchPath)
	if err != nil {
		if exists, _ := fs.Exists(patchPath); !exists {
			return nil, nil
		}
		return nil, microerror.Mask(err)
	}

	var patch struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Spec       struct {
			UserConfig struct {
				ConfigMap struct {
					Name string `json:"name"`
				} `json:"configMap"`
			} `json:"userConfig"`
		} `json:"spec"`
	}
	err = apiyaml.Unmarshal(data, &patch)
	if err != nil || !appcr.IsApp(patch.APIVersion, patch.Kind) {
		// A JSON 6902 patch is a list, so it does not unmarshal into an object:
		// the patch has already been migrated.
		return nil, nil
	}
	configMap := patch.Spec.UserConfig.ConfigMap.Name
	if configMap == "" {
		return nil, microerror.Maskf(unexpectedFileError, "%s is not the user config patch kubectl-gs generates, rewrite it as a patch adding to the HelmRelease's spec.valuesFrom by hand", rel(repoPath, patchPath))
	}

	kustomizationPath := filepath.Join(clusterDir, gitopskey.SigsKustomizationFileName())
	kustomization, err := readYAML(fs, kustomizationPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	entry := findPatch(kustomization, clusterUserConfigPatchFile)
	if entry == nil {
		return nil, microerror.Maskf(unexpectedFileError, "%s does not apply %s, add the patch by hand", rel(repoPath, kustomizationPath), clusterUserConfigPatchFile)
	}
	// A target the entry already has points at the App, so it is replaced.
	nameKey := scalar("name")
	nameKey.HeadComment = "A regular expression matching the name the base gives the HelmRelease\nbefore Flux substitutes the variables."
	deleteKey(entry, "target")
	entry.Content = append(entry.Content, scalar("target"), &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalar("group"), scalar("helm.toolkit.fluxcd.io"),
			scalar("kind"), scalar("HelmRelease"),
			nameKey, scalar(`\$\{cluster_name\}`),
		},
	})

	err = writeYAML(fs, kustomizationPath, kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newPatch := fmt.Sprintf("- op: add\n  path: /spec/valuesFrom/-\n  value:\n    kind: ConfigMap\n    name: %s\n    valuesKey: values\n", configMap)
	err = fs.WriteFile(patchPath, []byte(newPatch), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{
		fmt.Sprintf("%s: adds the user config to the HelmRelease's valuesFrom", rel(repoPath, patchPath)),
	}, nil
}

// removeClusterConfigPatch removes the patch pointing the Apps at the
// workload cluster's kubeconfig. HelmReleases name the kubeconfig themselves.
func removeClusterConfigPatch(fs *afero.Afero, repoPath, appsDir string) ([]string, error) {
	kustomizationPath := filepath.Join(appsDir, gitopskey.SigsKustomizationFileName())
	exists, err := fs.Exists(kustomizationPath)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}

	kustomization, err := readYAML(fs, kustomizationPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var changes []string

	patches := mapValue(kustomization, "patches")
	if entry := findPatch(kustomization, clusterConfigPatchFile); entry != nil {
		var kept []*yaml.Node
		for _, p := range patches.Content {
			if p != entry {
				kept = append(kept, p)
			}
		}
		patches.Content = kept
		if len(kept) == 0 {
			deleteKey(kustomization, "patches")
		}

		err = writeYAML(fs, kustomizationPath, kustomization)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, fmt.Sprintf("%s: removes the %s patch", rel(repoPath, kustomizationPath), clusterConfigPatchFile))
	}

	patchPath := filepath.Join(appsDir, clusterConfigPatchFile)
	exists, err = fs.Exists(patchPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if exists {
		err = fs.Remove(patchPath)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, fmt.Sprintf("%s: removed, there are no App CRs left to patch", rel(repoPath, patchPath)))
	}

	return changes, nil
}

// readYAML returns the top level mapping of the file.
func readYAML(fs *afero.Afero, path string) (*yaml.Node, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var doc yaml.Node
	err = yaml.Unmarshal(data, &doc)
	if err != nil {
		return nil, microerror.Maskf(unexpectedFileError, "cannot parse %s: %s", path, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, microerror.Maskf(unexpectedFileError, "%s is not a mapping", path)
	}

	return doc.Content[0], nil
}

func writeYAML(fs *afero.Afero, path string, node *yaml.Node) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err := encoder.Encode(node)
	if err != nil {
		return microerror.Mask(err)
	}
	err = encoder.Close()
	if err != nil {
		return microerror.Mask(err)
	}

	return microerror.Mask(fs.WriteFile(path, buf.Bytes(), 0644))
}

// findPatch returns the `patches` entry applying the given file.
func findPatch(kustomization *yaml.Node, file string) *yaml.Node {
	patches := mapValue(kustomization, "patches")
	if patches == nil || patches.Kind != yaml.SequenceNode {
		return nil
	}

	for _, p := range patches.Content {
		if path := mapValue(p, "path"); path != nil && path.Value == file {
			return p
		}
	}

	return nil
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func deleteKey(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}
