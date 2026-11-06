// Package appcr finds the App CRs a GitOps repository still deploys. Since
// repository structure version 2, kubectl-gs generates Flux HelmReleases
// instead, so a layer holding an App CR is at version 1.
package appcr

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
)

const kustomizationFile = "kustomization.yaml"

// MigrationDocs explains how to move clusters and apps from App CRs to Flux
// HelmReleases.
const MigrationDocs = "https://docs.giantswarm.io/overview/fleet-management/app-management/app-platform-deprecation/"

// Files returns the files holding an App CR that a Flux Kustomization
// pointed at dir deploys, relative to fs.
//
// When dir has a kustomization.yaml, only its resources are followed, files
// and directories alike, as kustomize would. Patches are not resources, so a
// patch targeting Apps does not count. Without one, every YAML file in dir and
// its subdirectories counts, as Flux generates a kustomization listing them.
// Resources outside fs, like remote bases, are skipped.
func Files(fs afero.Fs, dir string) ([]string, error) {
	found := map[string]bool{}
	err := walk(fs, dir, found, map[string]bool{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	files := make([]string, 0, len(found))
	for f := range found {
		files = append(files, f)
	}
	sort.Strings(files)

	return files, nil
}

func walk(fs afero.Fs, dir string, found, visited map[string]bool) error {
	dir = filepath.Clean(dir)
	if visited[dir] {
		return nil
	}
	visited[dir] = true

	resources, err := kustomizationResources(fs, dir)
	if os.IsNotExist(err) {
		entries, err := afero.ReadDir(fs, dir)
		if err != nil {
			return microerror.Mask(err)
		}
		for _, e := range entries {
			if e.IsDir() || isYAML(e.Name()) {
				resources = append(resources, e.Name())
			}
		}
	} else if err != nil {
		return microerror.Mask(err)
	}

	for _, r := range resources {
		path := filepath.Join(dir, r)
		info, err := fs.Stat(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		if info.IsDir() {
			err = walk(fs, path, found, visited)
			if err != nil {
				return microerror.Mask(err)
			}
			continue
		}

		ok, err := holdsApp(fs, path)
		if err != nil {
			return microerror.Mask(err)
		}
		if ok {
			found[path] = true
		}
	}

	return nil
}

func kustomizationResources(fs afero.Fs, dir string) ([]string, error) {
	data, err := afero.ReadFile(fs, filepath.Join(dir, kustomizationFile))
	if err != nil {
		return nil, err
	}

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	err = apiyaml.Unmarshal(data, &kustomization)
	if err != nil {
		return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filepath.Join(dir, kustomizationFile), err)
	}

	return kustomization.Resources, nil
}

func holdsApp(fs afero.Fs, path string) (bool, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return false, microerror.Mask(err)
	}

	decoder := apiyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var object struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
		err = decoder.Decode(&object)
		if err == io.EOF {
			return false, nil
		} else if err != nil {
			return false, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", path, err)
		}

		if IsApp(object.APIVersion, object.Kind) {
			return true, nil
		}
	}
}

// IsApp tells whether the type is a Giant Swarm App CR.
func IsApp(apiVersion, kind string) bool {
	return kind == "App" && strings.HasPrefix(apiVersion, "application.giantswarm.io/")
}

func isYAML(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}
