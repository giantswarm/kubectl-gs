package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"
)

// pathLists are the kustomization fields listing paths, and pathFields the
// list entries with a `path`.
var (
	pathLists  = []string{"resources", "components", "bases", "crds", "patchesStrategicMerge"}
	pathFields = []string{"patches", "patchesJson6902"}
	generators = []string{"configMapGenerator", "secretGenerator"}
)

// fixKustomizationPaths rewrites the relative paths of the kustomization.yaml
// moved from oldDir to newFile that lead out of movedDir, the directory it
// moved with, so they still point at the same files.
func fixKustomizationPaths(fs *afero.Afero, repoPath, movedDir, oldDir, newFile string) ([]string, error) {
	kustomization, err := readYAML(fs, newFile)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fix := func(p string) string {
		if p == "" || filepath.IsAbs(p) || strings.Contains(p, "://") || strings.HasPrefix(p, "github.com/") {
			return p
		}
		target := filepath.Clean(filepath.Join(oldDir, p))
		if target == movedDir || strings.HasPrefix(target, movedDir+string(filepath.Separator)) {
			return p
		}
		fixed, err := filepath.Rel(filepath.Dir(newFile), target)
		if err != nil {
			return p
		}
		return filepath.ToSlash(fixed)
	}

	if !rewritePaths(kustomization, fix) {
		return nil, nil
	}

	err = writeYAML(fs, newFile, kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: paths out of the cluster's directory fixed for its new place", rel(repoPath, newFile))}, nil
}

// rewritePaths applies fix to every path the kustomization names, and
// reports whether any changed.
func rewritePaths(kustomization *yaml.Node, fix func(string) string) bool {
	changed := false
	set := func(n *yaml.Node) {
		if n == nil || n.Kind != yaml.ScalarNode {
			return
		}
		if fixed := fix(n.Value); fixed != n.Value {
			n.Value = fixed
			changed = true
		}
	}

	for _, field := range pathLists {
		for _, n := range sequence(mapValue(kustomization, field)) {
			set(n)
		}
	}
	for _, field := range pathFields {
		for _, n := range sequence(mapValue(kustomization, field)) {
			set(mapValue(n, "path"))
		}
	}
	for _, field := range generators {
		for _, g := range sequence(mapValue(kustomization, field)) {
			rewriteGeneratorPaths(g, set)
		}
	}

	return changed
}

// rewriteGeneratorPaths fixes the files a generator reads, `key=path` ones
// included.
func rewriteGeneratorPaths(generator *yaml.Node, set func(*yaml.Node)) {
	for _, field := range []string{"files", "envs"} {
		for _, n := range sequence(mapValue(generator, field)) {
			key, path, found := strings.Cut(n.Value, "=")
			if !found {
				set(n)
				continue
			}
			value := &yaml.Node{Kind: yaml.ScalarNode, Value: path}
			set(value)
			n.Value = key + "=" + value.Value
		}
	}
	set(mapValue(generator, "env"))
}

func sequence(n *yaml.Node) []*yaml.Node {
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}

	return n.Content
}

// fixFluxPaths points the Flux Kustomizations next to the cluster's
// directory that applied its `mapi` directory at the cluster's directory.
func fixFluxPaths(fs *afero.Afero, repoPath, wcPath string) ([]string, error) {
	dir := filepath.Dir(filepath.Join(repoPath, wcPath))
	mapi := filepath.ToSlash(filepath.Join(wcPath, "mapi"))

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var changes []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" || isKustomizationFile(e.Name()) {
			continue
		}

		c, err := fixFluxPath(fs, repoPath, filepath.Join(dir, e.Name()), mapi, wcPath)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	return changes, nil
}

func fixFluxPath(fs *afero.Afero, repoPath, file, mapi, wcPath string) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil || !strings.Contains(string(data), mapi) {
		return nil, microerror.Mask(err)
	}

	kustomization, err := readYAML(fs, file)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	apiVersion, kind := mapValue(kustomization, "apiVersion"), mapValue(kustomization, "kind")
	if apiVersion == nil || kind == nil || kind.Value != "Kustomization" || !strings.HasPrefix(apiVersion.Value, "kustomize.toolkit.fluxcd.io/") {
		return nil, nil
	}

	spec := mapValue(kustomization, "spec")
	if spec == nil {
		return nil, nil
	}
	path := mapValue(spec, "path")
	if path == nil {
		return nil, nil
	}
	current := strings.TrimPrefix(path.Value, "./")
	if current != mapi && !strings.HasPrefix(current, mapi+"/") {
		return nil, nil
	}
	path.Value = "./" + wcPath + strings.TrimPrefix(current, mapi)

	err = writeYAML(fs, file, kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: applies %s", rel(repoPath, file), path.Value)}, nil
}
