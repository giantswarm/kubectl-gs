package migration

import (
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

// wholeDirectory returns what a cluster's directory holds besides
// `out-of-band`, for a cluster whose Kustomization applied all of it, and
// whether it has an `out-of-band` that leaves what the Kustomization applies.
// It refuses when a Kustomization that applies the directory prunes: Flux
// would delete from the management cluster what it applied from
// `out-of-band`.
func wholeDirectory(fs *afero.Afero, repoPath, wcDir string) ([]string, bool, error) {
	names, err := listDir(fs, wcDir)
	if err != nil {
		return nil, false, microerror.Mask(err)
	}

	files, _, err := mapiFiles(fs, repoPath, filepath.Join(wcDir, outOfBandDir), Env{})
	if err != nil || len(files) == 0 {
		return names, false, microerror.Mask(err)
	}

	pruning, err := pruningKustomizations(fs, repoPath, wcDir)
	if err != nil {
		return nil, false, microerror.Mask(err)
	}
	if len(pruning) > 0 {
		return nil, false, microerror.Maskf(unexpectedFileError, prunedOutOfBand, strings.Join(pruning, ", "), rel(repoPath, wcDir), rel(repoPath, filepath.Join(wcDir, outOfBandDir)))
	}

	return names, true, nil
}

const prunedOutOfBand = "%s applies %s with prune: true. The upgrade takes %s out of what it applies, so Flux would delete from the management cluster what it applied from there. Set prune: false on it, let Flux apply that, run the upgrade again and push it, and set prune: true again once Flux has applied the upgrade. What out-of-band applied to the management cluster stays there: delete it by hand if nothing needs it"

// pruningKustomizations returns the files next to the cluster's directory
// holding a Flux Kustomization that applies the directory with prune: true.
func pruningKustomizations(fs *afero.Afero, repoPath, wcDir string) ([]string, error) {
	dir := filepath.Dir(wcDir)
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var files []string
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if e.IsDir() || !isYAML(ext) || isKustomizationFile(e.Name()) {
			continue
		}

		prunes, err := prunesDirectory(fs, filepath.Join(dir, e.Name()), rel(repoPath, wcDir))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if prunes {
			files = append(files, rel(repoPath, filepath.Join(dir, e.Name())))
		}
	}

	return files, nil
}

// prunesDirectory reports whether a document of the file is a Flux
// Kustomization that applies dir with prune: true.
func prunesDirectory(fs *afero.Afero, file, dir string) (bool, error) {
	data, err := fs.ReadFile(file)
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, document := range documentSeparator.Split(string(data), -1) {
		if !strings.Contains(document, dir) {
			continue
		}
		object, err := decodeMapping([]byte(document), file)
		if err != nil {
			return false, microerror.Mask(err)
		}
		path := fluxPath(object)
		if path == nil {
			continue
		}
		prune := mapValue(mapValue(object, "spec"), "prune")
		if strings.TrimSuffix(strings.TrimPrefix(path.Value, "./"), "/") == dir && prune != nil && prune.Value == "true" {
			return true, nil
		}
	}

	return false, nil
}
