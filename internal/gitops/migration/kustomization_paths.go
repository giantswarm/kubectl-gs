package migration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"
)

// pathLists are the kustomization fields listing paths, and pathFields the
// list entries with a `path`.
var (
	pathLists  = []string{"resources", "components", "bases", "crds", "patchesStrategicMerge", "transformers", "generators", "validators"}
	pathFields = []string{"patches", "patchesJson6902", "replacements"}
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
		ext := filepath.Ext(e.Name())
		if e.IsDir() || !isYAML(ext) || isKustomizationFile(e.Name()) {
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

// documentSeparator splits a YAML stream into its documents.
var documentSeparator = regexp.MustCompile(`(?m)^---[ \t]*\n`)

// fixFluxPath rewrites the Flux Kustomizations in the file that apply
// `mapi`, document by document, and keeps every other document as it is.
func fixFluxPath(fs *afero.Afero, repoPath, file, mapi, wcPath string) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil || !strings.Contains(string(data), mapi) {
		return nil, microerror.Mask(err)
	}

	bounds := documentSeparator.FindAllIndex(data, -1)
	var out []byte
	var changes []string
	start := 0
	for i := 0; i <= len(bounds); i++ {
		end, next := len(data), len(data)
		if i < len(bounds) {
			end, next = bounds[i][0], bounds[i][1]
		}

		document, path, err := fixFluxDocument(data[start:end], file, mapi, wcPath)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if path != "" {
			changes = append(changes, fmt.Sprintf("%s: applies %s", rel(repoPath, file), path))
		}
		out = append(append(out, document...), data[end:next]...)
		start = next
	}

	if len(changes) == 0 {
		return nil, nil
	}

	return changes, microerror.Mask(fs.WriteFile(file, out, 0644))
}

// fixFluxDocument returns the document with the path of a Flux
// Kustomization applying `mapi` pointed at the cluster's directory, and that
// path, or the document as it is and an empty string.
func fixFluxDocument(document []byte, file, mapi, wcPath string) ([]byte, string, error) {
	if !strings.Contains(string(document), mapi) {
		return document, "", nil
	}

	kustomization, err := decodeMapping(document, file)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	path := fluxPath(kustomization)
	if path == nil {
		return document, "", nil
	}
	current := strings.TrimPrefix(path.Value, "./")
	if current != mapi && !strings.HasPrefix(current, mapi+"/") {
		return document, "", nil
	}
	path.Value = "./" + wcPath + strings.TrimPrefix(current, mapi)

	encoded, err := encodeYAML(kustomization)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	return encoded, path.Value, nil
}

// fluxPath returns the `spec.path` of a Flux Kustomization, or nil for any
// other object.
func fluxPath(object *yaml.Node) *yaml.Node {
	apiVersion, kind := mapValue(object, "apiVersion"), mapValue(object, "kind")
	if apiVersion == nil || kind == nil || kind.Value != "Kustomization" || !strings.HasPrefix(apiVersion.Value, "kustomize.toolkit.fluxcd.io/") {
		return nil
	}

	spec := mapValue(object, "spec")
	if spec == nil {
		return nil
	}

	return mapValue(spec, "path")
}

func isYAML(ext string) bool {
	return ext == ".yaml" || ext == ".yml"
}
