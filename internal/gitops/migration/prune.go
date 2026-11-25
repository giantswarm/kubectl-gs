package migration

import (
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const (
	pruneAnnotation = "kustomize.toolkit.fluxcd.io/prune"
	pruneDisabled   = "disabled"

	// upgradeDocs says how to upgrade a fork, and what to do first.
	upgradeDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/upgrade-fork.md#check-and-upgrade-the-structure-version"

	prunedMapi = "%s applies %s with prune: true, and %s builds objects without the " + pruneAnnotation + ": " + pruneDisabled + ` annotation. Objects that change path can be deleted and created again while they move. First give every object under mapi/ the annotation, e.g. with

  commonAnnotations:
    ` + pruneAnnotation + ": " + pruneDisabled + `

in each kustomization.yaml there, push, and wait for Flux to apply it. Run the upgrade in the next commit, and remove the annotations once the Kustomizations are Ready at that commit; keep it on the objects that carry it anyway, such as the cluster's HelmRelease. See ` + upgradeDocs
)

// kustomizationGenerators are the kustomization fields that build objects
// from something other than the resources it lists.
var kustomizationGenerators = []string{"components", "configMapGenerator", "generators", "helmCharts", "secretGenerator"}

// refusePrunedMapi fails for a cluster whose `mapi` a Flux Kustomization
// next to the cluster's directory applies with prune: true, unless every
// object it builds from there is annotated with
// kustomize.toolkit.fluxcd.io/prune: disabled, as the upgrade docs say.
func refusePrunedMapi(fs *afero.Afero, repoPath, wcDir string, env Env) error {
	mapi := filepath.ToSlash(rel(repoPath, filepath.Join(wcDir, gitopskey.MapiDirName())))

	dir := filepath.Dir(wcDir)
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, e := range entries {
		if e.IsDir() || !isYAML(filepath.Ext(e.Name())) || isKustomizationFile(e.Name()) {
			continue
		}

		err = refusePrunedFile(fs, repoPath, filepath.Join(dir, e.Name()), mapi, env)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// refusePrunedFile fails when a Flux Kustomization in the file applies mapi,
// or a directory in it, with prune: true, and builds an object there
// without the prune annotation.
func refusePrunedFile(fs *afero.Afero, repoPath, file, mapi string, env Env) error {
	paths, err := prunedPaths(fs, file)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, p := range paths {
		if p != mapi && !strings.HasPrefix(p, mapi+"/") {
			continue
		}

		unprotected, err := unprotectedIn(fs, repoPath, filepath.Join(repoPath, p), env, map[string]bool{})
		if err != nil {
			return microerror.Mask(err)
		}
		if unprotected != "" {
			return microerror.Maskf(unexpectedFileError, prunedMapi, rel(repoPath, file), p, unprotected)
		}
	}

	return nil
}

// prunedPaths returns the paths, without a leading `./` or a trailing `/`,
// of the Flux Kustomizations in the file that prune.
func prunedPaths(fs *afero.Afero, file string) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	documents, err := splitDocuments(data, file)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var paths []string
	for _, d := range documents {
		if d.node == nil {
			continue
		}
		path := fluxPath(d.node)
		if path == nil {
			continue
		}
		prune := mapValue(mapValue(d.node, "spec"), "prune")
		if prune != nil && prune.Value == "true" {
			paths = append(paths, strings.TrimSuffix(strings.TrimPrefix(path.Value, "./"), "/"))
		}
	}

	return paths, nil
}

// unprotectedIn returns the first file in the directory a Flux Kustomization
// applies that builds an object without the prune annotation, relative to
// the repository, or an empty string when every object has it. Without a
// kustomization file, Flux applies every YAML file there, and every
// directory with a kustomization file below it.
func unprotectedIn(fs *afero.Afero, repoPath, dir string, env Env, seen map[string]bool) (string, error) {
	kustomization, err := findKustomization(fs, dir)
	if err != nil {
		return "", microerror.Mask(err)
	}
	if kustomization != "" {
		return unprotectedKustomization(fs, repoPath, kustomization, env, seen)
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return "", microerror.Mask(err)
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())

		var unprotected string
		switch {
		case e.IsDir():
			unprotected, err = unprotectedIn(fs, repoPath, path, env, seen)
		case isYAML(filepath.Ext(e.Name())) && !env.IsIgnored(rel(repoPath, path)):
			unprotected, err = unprotectedFile(fs, repoPath, path)
		}
		if err != nil || unprotected != "" {
			return unprotected, microerror.Mask(err)
		}
	}

	return "", nil
}

// unprotectedKustomization returns the kustomization file, relative to the
// repository, unless it annotates what it builds, or every resource it
// lists is annotated. Anything it can't follow counts as not annotated.
func unprotectedKustomization(fs *afero.Afero, repoPath, path string, env Env, seen map[string]bool) (string, error) {
	if seen[path] {
		return "", nil
	}
	seen[path] = true

	kustomization, err := readYAML(fs, path)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if disablesPrune(mapValue(kustomization, "commonAnnotations")) {
		return "", nil
	}

	protected, err := resourcesProtected(fs, repoPath, path, kustomization, env, seen)
	if err != nil || protected {
		return "", microerror.Mask(err)
	}

	return rel(repoPath, path), nil
}

// resourcesProtected reports whether every resource the kustomization at
// path lists is annotated, and it generates nothing.
func resourcesProtected(fs *afero.Afero, repoPath, path string, kustomization *yaml.Node, env Env, seen map[string]bool) (bool, error) {
	for _, field := range kustomizationGenerators {
		if mapValue(kustomization, field) != nil {
			return false, nil
		}
	}

	for _, field := range []string{"resources", "bases", "crds"} {
		for _, r := range sequence(mapValue(kustomization, field)) {
			protected, err := resourceProtected(fs, repoPath, filepath.Join(filepath.Dir(path), r.Value), r.Value, env, seen)
			if err != nil || !protected {
				return false, microerror.Mask(err)
			}
		}
	}

	return true, nil
}

// resourceProtected reports whether every object the kustomization resource
// builds is annotated.
func resourceProtected(fs *afero.Afero, repoPath, target, entry string, env Env, seen map[string]bool) (bool, error) {
	if entry == "" || filepath.IsAbs(entry) || strings.Contains(entry, "://") || strings.HasPrefix(entry, "github.com/") {
		return false, nil
	}

	isDir, err := fs.DirExists(target)
	if err != nil {
		return false, microerror.Mask(err)
	}
	if isDir {
		kustomization, err := findKustomization(fs, target)
		if err != nil || kustomization == "" {
			return false, microerror.Mask(err)
		}
		unprotected, err := unprotectedKustomization(fs, repoPath, kustomization, env, seen)
		return unprotected == "", microerror.Mask(err)
	}

	exists, err := fs.Exists(target)
	if err != nil || !exists {
		return false, microerror.Mask(err)
	}
	unprotected, err := unprotectedFile(fs, repoPath, target)

	return unprotected == "", microerror.Mask(err)
}

// unprotectedFile returns the file, relative to the repository, unless every
// object in it is annotated.
func unprotectedFile(fs *afero.Afero, repoPath, path string) (string, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return "", microerror.Mask(err)
	}

	documents, err := splitDocuments(data, rel(repoPath, path))
	if err != nil {
		return "", microerror.Mask(err)
	}

	for _, d := range documents {
		if d.node != nil && !annotated(d.node) {
			return rel(repoPath, path), nil
		}
	}

	return "", nil
}

func annotated(object *yaml.Node) bool {
	metadata := mapValue(object, "metadata")
	if metadata == nil {
		return false
	}

	return disablesPrune(mapValue(metadata, "annotations"))
}

// disablesPrune reports whether the annotations disable pruning.
func disablesPrune(annotations *yaml.Node) bool {
	if annotations == nil {
		return false
	}
	v := mapValue(annotations, pruneAnnotation)

	return v != nil && v.Value == pruneDisabled
}
