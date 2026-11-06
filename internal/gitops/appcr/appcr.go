// Package appcr finds the App CRs a GitOps repository still deploys. Since
// repository structure version 2, kubectl-gs generates Flux HelmReleases
// instead, so a layer holding an App CR is at version 1.
package appcr

import (
	"bytes"
	"encoding/json"
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
func Files(fs afero.Fs, dir string) ([]string, error) {
	resources, err := Resources(fs, dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	files := []string{}
	for _, r := range resources {
		ok, err := holdsApp(fs, r)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if ok {
			files = append(files, r)
		}
	}

	return files, nil
}

// Resources returns the files a Flux Kustomization pointed at dir deploys,
// sorted and relative to fs.
//
// When dir has a kustomization.yaml, only its resources are followed, files
// and directories alike, as kustomize would. Patches are not resources, so a
// patch targeting Apps does not count. Without one, every YAML file in dir and
// its subdirectories counts, as Flux generates a kustomization listing them.
// Resources outside fs, like remote bases, are skipped.
func Resources(fs afero.Fs, dir string) ([]string, error) {
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

		found[path] = true
	}

	return nil
}

func kustomizationResources(fs afero.Fs, dir string) ([]string, error) {
	data, err := afero.ReadFile(fs, filepath.Join(dir, kustomizationFile))
	if err != nil {
		return nil, err
	}

	// Kustomize loads manifests from the deprecated `bases` and from
	// `components` too.
	var kustomization struct {
		Bases      []string `json:"bases"`
		Components []string `json:"components"`
		Resources  []string `json:"resources"`
	}
	err = apiyaml.Unmarshal(data, &kustomization)
	if err != nil {
		return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filepath.Join(dir, kustomizationFile), err)
	}

	resources := append(kustomization.Resources, kustomization.Bases...)
	return append(resources, kustomization.Components...), nil
}

// Patches returns the patch files the kustomization.yaml in dir applies to
// App CRs, relative to fs. A patch counts when it targets the App kind or is
// an App itself. An inline patch targeting Apps is reported as the
// kustomization.yaml. Unlike Files, it doesn't follow resources.
func Patches(fs afero.Fs, dir string) ([]string, error) {
	kustomizationPath := filepath.Join(dir, kustomizationFile)
	data, err := afero.ReadFile(fs, kustomizationPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	var kustomization struct {
		Patches []struct {
			Path   string `json:"path"`
			Target struct {
				Kind string `json:"kind"`
			} `json:"target"`
		} `json:"patches"`
		PatchesStrategicMerge []string `json:"patchesStrategicMerge"`
	}
	err = apiyaml.Unmarshal(data, &kustomization)
	if err != nil {
		return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", kustomizationPath, err)
	}

	paths := kustomization.PatchesStrategicMerge
	found := []string{}
	for _, p := range kustomization.Patches {
		switch {
		case p.Target.Kind == "App" && p.Path == "":
			found = append(found, kustomizationPath)
		case p.Target.Kind == "App":
			found = append(found, filepath.Join(dir, p.Path))
		case p.Path != "":
			paths = append(paths, p.Path)
		}
	}

	for _, p := range paths {
		path := filepath.Join(dir, p)
		ok, err := holdsApp(fs, path)
		if os.IsNotExist(microerror.Cause(err)) {
			continue
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
		if ok {
			found = append(found, path)
		}
	}

	return found, nil
}

func holdsApp(fs afero.Fs, path string) (bool, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return false, microerror.Mask(err)
	}
	documents, err := Documents(data, path)
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, document := range documents {
		var object struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
		err = json.Unmarshal(document, &object)
		if err != nil {
			return false, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", path, err)
		}

		if IsApp(object.APIVersion, object.Kind) {
			return true, nil
		}
	}

	return false, nil
}

// Documents returns the YAML or JSON documents in data that are objects. A
// JSON 6902 patch file is a list of operations, not a resource, and empty
// documents hold nothing, so both are left out. path only names the file in
// errors.
func Documents(data []byte, path string) ([]json.RawMessage, error) {
	var documents []json.RawMessage
	decoder := apiyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var document json.RawMessage
		err := decoder.Decode(&document)
		if err == io.EOF {
			return documents, nil
		} else if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", path, err)
		}

		trimmed := bytes.TrimSpace(document)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] == '[' {
			continue
		}
		documents = append(documents, document)
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
