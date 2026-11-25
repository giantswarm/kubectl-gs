package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const outOfBandDir = "out-of-band"

// migrateLayout moves a workload cluster's `mapi` directory up into the
// cluster's directory, and adds the `kustomization.yaml` that lists what the
// cluster's Kustomization applies, never `out-of-band`.
func migrateLayout(fs *afero.Afero, repoPath, wcPath string, env Env) ([]string, error) {
	wcDir := filepath.Join(repoPath, wcPath)

	err := checkMapiMove(fs, repoPath, wcDir, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	moved, changes, err := moveMapiUp(fs, repoPath, wcDir, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c, err := addClusterKustomization(fs, repoPath, wcDir, moved, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	changes = append(changes, c...)

	if len(moved) > 0 {
		c, err = fixFluxPaths(fs, repoPath, wcPath)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	return changes, nil
}

// checkMapiMove refuses, before anything moves, a cluster whose `mapi` can't
// move up safely.
func checkMapiMove(fs *afero.Afero, repoPath, wcDir string, env Env) error {
	files, _, err := mapiFiles(fs, repoPath, filepath.Join(wcDir, gitopskey.MapiDirName()), env)
	if err != nil || len(files) == 0 {
		return microerror.Mask(err)
	}

	err = refuseClusterKustomization(fs, repoPath, wcDir, topLevel(files))
	if err != nil {
		return microerror.Mask(err)
	}

	err = refusePrunedMapi(fs, repoPath, wcDir, env)
	if err != nil {
		return microerror.Mask(err)
	}

	return microerror.Mask(refuseSopsRules(fs, repoPath, wcDir, files))
}

// refuseClusterKustomization fails for a cluster with a kustomization file
// of its own next to `mapi`: it doesn't list what moves up from there, so
// the cluster's Kustomization would stop applying it, and, pruning, delete
// it.
func refuseClusterKustomization(fs *afero.Afero, repoPath, wcDir string, moving []string) error {
	path, err := findKustomization(fs, wcDir)
	if err != nil || path == "" {
		return microerror.Mask(err)
	}

	return microerror.Maskf(unexpectedFileError, "%s exists next to mapi/, and doesn't list %s, which the upgrade moves up from mapi/: the cluster's Kustomization would stop applying them, and delete them if it prunes. Move what mapi/ holds up and list it there by hand, or remove %s, then run the upgrade again", rel(repoPath, path), strings.Join(moving, ", "), filepath.Base(path))
}

// moveMapiUp moves what `mapi` holds into the cluster's directory, but for
// the files git ignores, and returns the names it moved there.
func moveMapiUp(fs *afero.Afero, repoPath, wcDir string, env Env) ([]string, []string, error) {
	mapiDir := filepath.Join(wcDir, gitopskey.MapiDirName())

	files, left, err := mapiFiles(fs, repoPath, mapiDir, env)
	if err != nil || len(files) == 0 {
		return nil, nil, microerror.Mask(err)
	}

	for _, f := range files {
		to := filepath.Join(wcDir, f)
		exists, err := fs.Exists(to)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		if exists {
			return nil, nil, microerror.Maskf(unexpectedFileError, "both %s and %s exist, merge them by hand first", rel(repoPath, to), rel(repoPath, filepath.Join(mapiDir, f)))
		}
	}

	var changes []string
	for _, f := range files {
		c, err := moveFile(fs, repoPath, mapiDir, wcDir, f)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	err = removeEmptyDirs(fs, mapiDir)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	moved := topLevel(files)
	for _, m := range moved {
		changes = append([]string{fmt.Sprintf("%s: moved up from mapi/", rel(repoPath, filepath.Join(wcDir, m)))}, changes...)
	}
	for _, l := range left {
		changes = append(changes, fmt.Sprintf("%s: left where it is, as git ignores it. Move it yourself", l))
	}

	return moved, changes, nil
}

// mapiFiles lists the regular files under mapiDir, relative to it, and the
// ones git ignores, relative to the repository.
func mapiFiles(fs *afero.Afero, repoPath, mapiDir string, env Env) ([]string, []string, error) {
	exists, err := fs.DirExists(mapiDir)
	if err != nil || !exists {
		return nil, nil, microerror.Mask(err)
	}

	var files, ignored []string
	err = fs.Walk(mapiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.Mode().IsRegular() {
			return err
		}
		if env.IsIgnored(rel(repoPath, path)) {
			ignored = append(ignored, rel(repoPath, path))
			return nil
		}
		files = append(files, rel(mapiDir, path))
		return nil
	})

	return files, ignored, microerror.Mask(err)
}

// moveFile moves the file at f under from to the same place under to, and
// fixes the paths of a kustomization.yaml that lead out of from.
func moveFile(fs *afero.Afero, repoPath, from, to, f string) ([]string, error) {
	src, dst := filepath.Join(from, f), filepath.Join(to, f)

	err := fs.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	err = fs.Rename(src, dst)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if !isKustomizationFile(f) {
		return nil, nil
	}

	return fixKustomizationPaths(fs, repoPath, from, filepath.Dir(src), dst)
}

// kustomizationFileNames are the names kustomize reads a kustomization from,
// in the order it looks for them.
var kustomizationFileNames = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}

func isKustomizationFile(path string) bool {
	return slices.Contains(kustomizationFileNames, filepath.Base(path))
}

// findKustomization returns the path of the kustomization file in dir, by
// any name kustomize reads, or an empty string when it has none.
func findKustomization(fs *afero.Afero, dir string) (string, error) {
	for _, name := range kustomizationFileNames {
		path := filepath.Join(dir, name)
		exists, err := fs.Exists(path)
		if err != nil {
			return "", microerror.Mask(err)
		}
		if exists {
			return path, nil
		}
	}

	return "", nil
}

// removeEmptyDirs removes dir and the directories below it that hold no
// file.
func removeEmptyDirs(fs *afero.Afero, dir string) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, e := range entries {
		if e.IsDir() {
			err = removeEmptyDirs(fs, filepath.Join(dir, e.Name()))
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	empty, err := afero.IsEmpty(fs, dir)
	if err != nil || !empty {
		return microerror.Mask(err)
	}

	return microerror.Mask(fs.Remove(dir))
}

// topLevel returns the first path segments of the files, sorted.
func topLevel(files []string) []string {
	seen := map[string]bool{}
	for _, f := range files {
		seen[strings.SplitN(filepath.ToSlash(f), "/", 2)[0]] = true
	}

	var out []string
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)

	return out
}

// addClusterKustomization writes the cluster's `kustomization.yaml`, when it
// has no kustomization file, by any name kustomize reads. It lists what moved up from `mapi`, which the cluster's
// Kustomization applied, or else what the cluster's directory holds, which
// it applied all of, but `out-of-band`.
func addClusterKustomization(fs *afero.Afero, repoPath, wcDir string, moved []string, env Env) ([]string, error) {
	existing, err := findKustomization(fs, wcDir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if existing != "" {
		return nil, refuseOutOfBand(fs, repoPath, existing)
	}
	path := filepath.Join(wcDir, gitopskey.SigsKustomizationFileName())

	names := moved
	outOfBand := false
	if len(names) == 0 {
		names, outOfBand, err = wholeDirectory(fs, repoPath, wcDir)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources, err := resourcesOf(fs, repoPath, wcDir, names, env, "the cluster's kustomization.yaml")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = fs.WriteFile(path, []byte(clusterKustomization(resources)), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	changes := []string{fmt.Sprintf("%s: added, lists %s", rel(repoPath, path), strings.Join(resources, ", "))}
	if outOfBand {
		changes = append(changes, fmt.Sprintf("%s: no longer applied to the management cluster. The cluster's Kustomization doesn't prune, so what it applied from there stays: delete it by hand if nothing needs it", rel(repoPath, filepath.Join(wcDir, outOfBandDir))))
	}

	return changes, nil
}

func clusterKustomization(resources []string) string {
	var b strings.Builder
	b.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n")
	b.WriteString("# What the cluster's Kustomizations apply. Never list out-of-band/ here: what\n# it holds goes straight into the workload cluster, never into the management\n# cluster.\n")
	if len(resources) == 0 {
		b.WriteString("resources: []\n")
		return b.String()
	}
	b.WriteString("resources:\n")
	for _, r := range resources {
		fmt.Fprintf(&b, "  - %s\n", r)
	}

	return b.String()
}

// listDir returns the names in dir, but for `out-of-band`, `mapi`, hidden
// ones and the kustomization.yaml.
func listDir(fs *afero.Afero, dir string) ([]string, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var names []string
	for _, e := range entries {
		switch {
		case e.Name() == outOfBandDir, e.Name() == gitopskey.MapiDirName(), strings.HasPrefix(e.Name(), "."), isKustomizationFile(e.Name()):
			continue
		}
		names = append(names, e.Name())
	}

	return names, nil
}

// resourcesOf returns the kustomization resources for the names in dir: a
// directory with a kustomization.yaml as `name/`, and a YAML file as it is.
// It fails on a directory with files but no kustomization.yaml, which only
// Flux's generated kustomization could apply.
func resourcesOf(fs *afero.Afero, repoPath, dir string, names []string, env Env, owner string) ([]string, error) {
	var resources []string
	for _, name := range names {
		path := filepath.Join(dir, name)

		resource, err := resourceOf(fs, repoPath, path, env, owner)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if resource != "" {
			resources = append(resources, resource)
		}
	}
	sort.Strings(resources)

	return resources, nil
}

func resourceOf(fs *afero.Afero, repoPath, path string, env Env, owner string) (string, error) {
	isDir, err := fs.DirExists(path)
	if err != nil {
		return "", microerror.Mask(err)
	}
	if !isDir {
		ext := filepath.Ext(path)
		if isYAML(ext) && !env.IsIgnored(rel(repoPath, path)) {
			return filepath.Base(path), nil
		}
		return "", nil
	}

	kustomization, err := findKustomization(fs, path)
	if err != nil {
		return "", microerror.Mask(err)
	}
	if kustomization != "" {
		return filepath.Base(path) + "/", nil
	}

	files, _, err := mapiFiles(fs, repoPath, path, env)
	if err != nil || len(files) == 0 {
		return "", microerror.Mask(err)
	}

	return "", microerror.Maskf(unexpectedFileError, "%s has no kustomization.yaml, so %s can't list it. Add one first", rel(repoPath, path), owner)
}

// refuseOutOfBand fails for a cluster's kustomization.yaml that lists
// `out-of-band`, which must never be applied to the management cluster.
func refuseOutOfBand(fs *afero.Afero, repoPath, path string) error {
	kustomization, err := readYAML(fs, path)
	if err != nil {
		return microerror.Mask(err)
	}

	resources := mapValue(kustomization, "resources")
	if resources == nil || resources.Kind != yaml.SequenceNode {
		return nil
	}
	for _, r := range resources.Content {
		if strings.SplitN(strings.TrimPrefix(r.Value, "./"), "/", 2)[0] == outOfBandDir {
			return microerror.Maskf(unexpectedFileError, "%s lists %s, which goes straight into the workload cluster and must not be applied to the management cluster. Remove it first", rel(repoPath, path), r.Value)
		}
	}

	return nil
}
