package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	clusterUserConfigPatchFile = "patch_cluster_userconfig.yaml"
	clusterConfigPatchFile     = "patch_cluster_config.yaml"

	appPatchHint = "Rewrite them for the HelmRelease, or remove them, see " + appcr.MigrationDocs
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

func applyHelmReleases(fs *afero.Afero, repoPath string, layer metadata.Layer, env Env) ([]string, error) {
	dir := filepath.Join(repoPath, layer.Path)

	switch layer.Kind {
	case metadata.LayerClusterBase:
		err := refuseAppCRs(fs, repoPath, dir, "Move the cluster to a release-<provider> HelmRelease first, see "+appcr.MigrationDocs)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		return nil, refuseAppPatches(fs, repoPath, dir, appPatchHint)
	case metadata.LayerApp:
		err := refuseAppCRs(fs, repoPath, dir, "Move the app to a HelmRelease first, see "+appcr.MigrationDocs)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		return nil, refuseAppPatches(fs, repoPath, dir, "Rewrite them as JSON 6902 patches adding to the HelmRelease's spec.valuesFrom, see "+appcr.MigrationDocs)
	case metadata.LayerWorkloadCluster:
		return migrateWorkloadCluster(fs, repoPath, dir, env)
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

// refuseAppPatches fails for a layer that still patches App CRs, like the user
// values patch `gitops add app --base` generated. Once the App is a
// HelmRelease, Kustomize fails on a patch that matches nothing.
func refuseAppPatches(fs *afero.Afero, repoPath, dir, hint string) error {
	files, err := appcr.Patches(fs, dir)
	if err != nil {
		return microerror.Mask(err)
	}
	if len(files) == 0 {
		return nil
	}

	for i, f := range files {
		files[i] = rel(repoPath, f)
	}

	return microerror.Maskf(appCRError, "%s still patches App CRs:\n  %s\n%s", rel(repoPath, dir), strings.Join(files, "\n  "), hint)
}

func migrateWorkloadCluster(fs *afero.Afero, repoPath, wcDir string, env Env) ([]string, error) {
	// Clusters added before structure version 3 without `--skip-mapi` keep
	// their manifests in `mapi`.
	dir := filepath.Join(wcDir, gitopskey.MapiDirName())
	exists, err := fs.DirExists(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !exists {
		dir = wcDir
	}

	changes, err := migrateDir(fs, repoPath, filepath.Join(dir, gitopskey.ClusterDirName()), env, "Move the cluster to a release-<provider> HelmRelease first, see "+appcr.MigrationDocs, migrateClusterUserConfigPatch)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c, err := migrateDir(fs, repoPath, filepath.Join(dir, gitopskey.AppsDirName()), env, "Move the apps to HelmReleases first, see "+appcr.MigrationDocs, removeClusterConfigPatch)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return append(changes, c...), nil
}

// migrateDir migrates a directory of a workload cluster, when it exists: it
// refuses App CRs there, with the hint, rewrites what kubectl-gs generated
// around them with migrate, and refuses the App patches left.
func migrateDir(fs *afero.Afero, repoPath, dir string, env Env, hint string, migrate func(*afero.Afero, string, string, Env) ([]string, error)) ([]string, error) {
	exists, err := fs.DirExists(dir)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}

	err = refuseAppCRs(fs, repoPath, dir, hint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	changes, err := migrate(fs, repoPath, dir, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return changes, microerror.Mask(refuseAppPatches(fs, repoPath, dir, appPatchHint))
}

// migrateClusterUserConfigPatch turns the patch adding the user config to the
// cluster App into one adding it to the HelmRelease's valuesFrom. A merge
// patch would replace the base's valuesFrom list, so it becomes a JSON 6902
// patch, which needs a target. Neither file is changed when git ignores
// either: git cannot restore it, so the user rewrites the patch by hand.
func migrateClusterUserConfigPatch(fs *afero.Afero, repoPath, clusterDir string, env Env) ([]string, error) {
	patchPath := filepath.Join(clusterDir, clusterUserConfigPatchFile)
	data, err := fs.ReadFile(patchPath)
	if err != nil {
		if exists, _ := fs.Exists(patchPath); !exists {
			return nil, nil
		}
		return nil, microerror.Mask(err)
	}

	var patch map[string]any
	err = apiyaml.Unmarshal(data, &patch)
	if err != nil || !appcr.IsApp(stringField(patch, "apiVersion"), stringField(patch, "kind")) {
		// A JSON 6902 patch is a list, so it does not unmarshal into an object:
		// the patch has already been migrated.
		return nil, nil
	}
	// Only the patch kubectl-gs generates is rewritten. Anything else it sets,
	// like a user config Secret, would be lost.
	spec, _ := patch["spec"].(map[string]any)
	userConfig, _ := spec["userConfig"].(map[string]any)
	configMapRef, _ := userConfig["configMap"].(map[string]any)
	configMap := stringField(configMapRef, "name")
	if configMap == "" || len(spec) != 1 || len(userConfig) != 1 {
		return nil, microerror.Maskf(unexpectedFileError, "%s is not the user config patch kubectl-gs generates, rewrite it as a patch adding to the HelmRelease's spec.valuesFrom by hand", rel(repoPath, patchPath))
	}

	return retargetClusterUserConfigPatch(fs, repoPath, clusterDir, configMap, env)
}

// retargetClusterUserConfigPatch rewrites the user config patch as a JSON
// 6902 patch adding the ConfigMap to the HelmRelease's valuesFrom, and gives
// its entry in the cluster's kustomization.yaml the HelmRelease as target. It
// refuses when git ignores either file.
func retargetClusterUserConfigPatch(fs *afero.Afero, repoPath, clusterDir, configMap string, env Env) ([]string, error) {
	patchPath := filepath.Join(clusterDir, clusterUserConfigPatchFile)
	kustomizationPath := filepath.Join(clusterDir, gitopskey.SigsKustomizationFileName())
	for _, p := range []string{patchPath, kustomizationPath} {
		if env.IsIgnored(rel(repoPath, p)) {
			return nil, microerror.Maskf(unexpectedFileError, "%s has to become a patch adding to the HelmRelease's spec.valuesFrom, but git ignores %s, so the upgrade leaves it: rewrite the patch by hand, see %s", rel(repoPath, patchPath), rel(repoPath, p), appcr.MigrationDocs)
		}
	}
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
// A patch already rewritten to target HelmReleases is kept, as the
// HelmReleases may rely on it to reach the workload cluster. A
// kustomization.yaml git ignores is never edited, as git cannot restore it,
// and a patch git ignores is left where it is.
func removeClusterConfigPatch(fs *afero.Afero, repoPath, appsDir string, env Env) ([]string, error) {
	kustomizationPath := filepath.Join(appsDir, gitopskey.SigsKustomizationFileName())
	exists, err := fs.Exists(kustomizationPath)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}

	kustomization, err := readYAML(fs, kustomizationPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patchPath := filepath.Join(appsDir, clusterConfigPatchFile)
	patchExists, patchIsApp, err := readAppPatch(fs, patchPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	entry := findPatch(kustomization, clusterConfigPatchFile)
	generatedEntry := entry == nil || patchTargetKind(entry) == "App"
	generatedPatch := !patchExists || patchIsApp
	switch {
	case entry == nil && !patchExists:
		return nil, nil
	case !generatedEntry && !generatedPatch:
		return nil, nil
	case !generatedEntry || !generatedPatch:
		return nil, microerror.Maskf(unexpectedFileError, "%s is not the patch kubectl-gs generates to point App CRs at the workload cluster, remove it or rewrite it for HelmReleases by hand", rel(repoPath, patchPath))
	}

	changes, err := dropPatchEntry(fs, repoPath, kustomizationPath, kustomization, entry, env)
	if err != nil || !patchExists {
		return changes, microerror.Mask(err)
	}

	c, err := removePatchFile(fs, repoPath, patchPath, env)

	return append(changes, c...), microerror.Mask(err)
}

// readAppPatch reports whether the patch at path exists, and whether it
// patches an App CR.
func readAppPatch(fs *afero.Afero, path string) (bool, bool, error) {
	exists, err := fs.Exists(path)
	if err != nil || !exists {
		return false, false, microerror.Mask(err)
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return false, false, microerror.Mask(err)
	}
	var patch map[string]any
	isApp := apiyaml.Unmarshal(data, &patch) == nil && appcr.IsApp(stringField(patch, "apiVersion"), stringField(patch, "kind"))

	return true, isApp, nil
}

// dropPatchEntry removes the entry from the patches of the kustomization.yaml
// at path, unless it is nil. It refuses when git ignores the file, as git
// cannot restore it.
func dropPatchEntry(fs *afero.Afero, repoPath, path string, kustomization, entry *yaml.Node, env Env) ([]string, error) {
	if entry == nil {
		return nil, nil
	}
	if env.IsIgnored(rel(repoPath, path)) {
		return nil, microerror.Maskf(unexpectedFileError, "%s applies %s, which patches App CRs, but git ignores it, so the upgrade leaves it: remove the patch from it by hand", rel(repoPath, path), clusterConfigPatchFile)
	}

	patches := mapValue(kustomization, "patches")
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

	err := writeYAML(fs, path, kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: removes the %s patch", rel(repoPath, path), clusterConfigPatchFile)}, nil
}

// removePatchFile removes the patch at path, which patches nothing any more,
// unless git ignores it: git cannot restore it, so it is left where it is.
func removePatchFile(fs *afero.Afero, repoPath, path string, env Env) ([]string, error) {
	if env.IsIgnored(rel(repoPath, path)) {
		return []string{fmt.Sprintf("%s: left where it is, as git ignores it. Remove it yourself", rel(repoPath, path))}, nil
	}

	err := fs.Remove(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: removed, there are no App CRs left to patch", rel(repoPath, path))}, nil
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

// patchTargetKind returns the kind a `patches` entry targets.
func patchTargetKind(entry *yaml.Node) string {
	target := mapValue(entry, "target")
	if target == nil {
		return ""
	}
	kind := mapValue(target, "kind")
	if kind == nil {
		return ""
	}

	return kind.Value
}

func stringField(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
