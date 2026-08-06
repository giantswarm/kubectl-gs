package metadata

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/pkg/project"
)

// header is prepended on every write. The file is fully re-marshalled each
// time, so the comment has to be re-added rather than preserved.
const header = `# This file is managed by ` + "`kubectl gs gitops`" + `. Do not edit it by hand.
# It records which version of the repository structure each part of this
# repository was generated with, so that ` + "`kubectl gs gitops check`" + ` can tell
# you when the repository has fallen behind.
`

// New returns empty metadata stamped with the current structure version.
func New() *RepositoryMetadata {
	return &RepositoryMetadata{
		APIVersion:       APIVersion,
		Kind:             Kind,
		StructureVersion: StructureVersion,
		GeneratedWith:    GeneratedWith(),
		Layers:           []Layer{},
	}
}

// GeneratedWith identifies the kubectl-gs build in use, e.g. `kubectl-gs/5.7.3`.
func GeneratedWith() string {
	return fmt.Sprintf("%s/%s", project.Name(), project.Version())
}

// FilePath returns the path to the metadata file of the repository cloned at
// repoPath.
func FilePath(repoPath string) string {
	return filepath.Join(repoPath, FileName)
}

// Load reads the metadata of the repository cloned at repoPath. It returns a
// NotFoundError if the repository holds no metadata yet, which is the case for
// every repository created before `kubectl gs gitops` started recording it.
func Load(fs *afero.Afero, repoPath string) (*RepositoryMetadata, error) {
	path := FilePath(repoPath)

	exists, err := fs.Exists(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !exists {
		return nil, microerror.Maskf(NotFoundError, "no %s found in %s", FileName, repoPath)
	}

	rawYaml, err := fs.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	md := &RepositoryMetadata{}
	err = yaml.Unmarshal(rawYaml, md)
	if err != nil {
		return nil, microerror.Maskf(invalidMetadataError, "%s is not valid YAML: %s", path, err)
	}

	if md.Kind != Kind {
		return nil, microerror.Maskf(invalidMetadataError, "%s holds kind %q, expected %q", path, md.Kind, Kind)
	}
	if md.APIVersion != APIVersion {
		return nil, microerror.Maskf(invalidMetadataError, "%s holds apiVersion %q, expected %q", path, md.APIVersion, APIVersion)
	}

	return md, nil
}

// LoadOrNew reads the repository metadata, falling back to empty metadata when
// the repository holds none yet. Errors other than a missing file are returned,
// so that a corrupted file is never silently replaced.
func LoadOrNew(fs *afero.Afero, repoPath string) (*RepositoryMetadata, error) {
	md, err := Load(fs, repoPath)
	if IsNotFound(err) {
		return New(), nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return md, nil
}

// Render marshals the metadata into the bytes to write to disk.
func Render(md *RepositoryMetadata) ([]byte, error) {
	md.Sort()

	body, err := yaml.Marshal(md)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return append([]byte(header), body...), nil
}

// Save writes the metadata to the repository cloned at repoPath, replacing what
// is already there. Unlike the generated structure, this file is ours to own.
func Save(fs *afero.Afero, repoPath string, md *RepositoryMetadata) error {
	body, err := Render(md)
	if err != nil {
		return microerror.Mask(err)
	}

	// 0600 to match the permissions the creator gives every other generated
	// file, see creator.defaultFilePerm.
	err = fs.WriteFile(FilePath(repoPath), body, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Upsert records a layer, replacing the entry for the same kind and path if
// there already is one. Re-running an `add` command therefore refreshes the
// entry instead of duplicating it.
func (md *RepositoryMetadata) Upsert(layer Layer) {
	if layer.StructureVersion == 0 {
		layer.StructureVersion = StructureVersion
	}
	if layer.GeneratedWith == "" {
		layer.GeneratedWith = GeneratedWith()
	}

	for i, l := range md.Layers {
		if l.Kind == layer.Kind && l.Path == layer.Path {
			md.Layers[i] = layer
			return
		}
	}

	md.Layers = append(md.Layers, layer)
}

// Has reports whether a layer is already recorded.
func (md *RepositoryMetadata) Has(kind, path string) bool {
	for _, l := range md.Layers {
		if l.Kind == kind && l.Path == path {
			return true
		}
	}

	return false
}

// MergeMissing records the given layers that are not tracked yet and returns
// them. Layers already recorded are left untouched, so that merging never
// bumps a structure version that was legitimately recorded as older.
func (md *RepositoryMetadata) MergeMissing(layers []Layer) []Layer {
	added := []Layer{}
	for _, l := range layers {
		if md.Has(l.Kind, l.Path) {
			continue
		}

		md.Upsert(l)
		added = append(added, l)
	}

	return added
}

// Stamp records that the current kubectl-gs has written to the repository. The
// repository level structure version only ever moves forward, so that a repo
// touched by an older kubectl-gs is not reported as having gone backwards.
func (md *RepositoryMetadata) Stamp() {
	md.APIVersion = APIVersion
	md.Kind = Kind
	md.GeneratedWith = GeneratedWith()

	if md.StructureVersion < StructureVersion {
		md.StructureVersion = StructureVersion
	}
}

// Sort orders the layers by path, so that the file reads like the repository
// tree and produces a stable diff.
func (md *RepositoryMetadata) Sort() {
	sort.SliceStable(md.Layers, func(i, j int) bool {
		if md.Layers[i].Path != md.Layers[j].Path {
			return md.Layers[i].Path < md.Layers[j].Path
		}
		return md.Layers[i].Kind < md.Layers[j].Kind
	})
}

// OutdatedLayers returns the layers generated with an older structure version
// than the one this kubectl-gs produces.
func (md *RepositoryMetadata) OutdatedLayers() []Layer {
	outdated := []Layer{}
	for _, l := range md.Layers {
		if l.StructureVersion < StructureVersion {
			outdated = append(outdated, l)
		}
	}

	return outdated
}
