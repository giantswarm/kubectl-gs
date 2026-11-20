package metadata

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

const (
	appCR          = "apiVersion: application.giantswarm.io/v1alpha1\nkind: App\n"
	helmRelease    = "apiVersion: helm.toolkit.fluxcd.io/v2\nkind: HelmRelease\n"
	baseResources  = "resources:\n  - cluster.yaml\n"
	capaBase       = "bases/clusters/capa"
	capaLegacyBase = capaBase + "/template"
)

type clusterBaseLayoutCase struct {
	name            string
	dirs            []string
	files           map[string]string
	expectedLayers  map[string]int
	expectedSkipped []string
}

// clusterBaseLayoutCases cover both places `gitops add base` has written a
// cluster base: `bases/clusters/PROVIDER`, and
// `bases/clusters/PROVIDER/template` before it. Repositories in either layout,
// or a mix of both, are adopted.
var clusterBaseLayoutCases = []clusterBaseLayoutCase{
	{
		name: "base in the provider directory",
		files: map[string]string{
			capaBase + "/kustomization.yaml": baseResources,
			capaBase + "/cluster.yaml":       helmRelease,
		},
		expectedLayers:  map[string]int{capaBase: StructureVersion},
		expectedSkipped: []string{},
	},
	{
		name: "base in the template directory of the earlier layout",
		files: map[string]string{
			capaLegacyBase + "/kustomization.yaml": baseResources,
			capaLegacyBase + "/cluster.yaml":       helmRelease,
		},
		expectedLayers:  map[string]int{capaLegacyBase: StructureVersion},
		expectedSkipped: []string{},
	},
	{
		name:            "empty template directory of the earlier layout",
		dirs:            []string{capaLegacyBase},
		expectedLayers:  map[string]int{capaLegacyBase: StructureVersion},
		expectedSkipped: []string{},
	},
	{
		name: "both layouts in one repository",
		files: map[string]string{
			capaBase + "/kustomization.yaml":                  baseResources,
			capaBase + "/cluster.yaml":                        helmRelease,
			"bases/clusters/capz/template/kustomization.yaml": baseResources,
			"bases/clusters/capz/template/cluster.yaml":       appCR,
		},
		expectedLayers: map[string]int{
			capaBase:                       StructureVersion,
			"bases/clusters/capz/template": 1,
		},
		expectedSkipped: []string{},
	},
	{
		name: "App CR in the provider directory",
		files: map[string]string{
			capaBase + "/kustomization.yaml": baseResources,
			capaBase + "/cluster.yaml":       appCR,
		},
		expectedLayers:  map[string]int{capaBase: 1},
		expectedSkipped: []string{},
	},
	{
		name:            "provider directory without a base",
		dirs:            []string{"bases/clusters/capv/notes"},
		files:           map[string]string{"bases/clusters/capv/readme.md": "# Not a base\n"},
		expectedLayers:  map[string]int{},
		expectedSkipped: []string{"bases/clusters/capv"},
	},
}

func Test_Adopt_ClusterBaseLayouts(t *testing.T) {
	for _, tc := range clusterBaseLayoutCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Adopt(layoutRepo(t, tc.dirs, tc.files), testRepo)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(tc.expectedLayers, clusterBaseVersions(t, result)); diff != "" {
				t.Errorf("adopted cluster bases differ (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.expectedSkipped, result.Skipped); diff != "" {
				t.Errorf("skipped directories differ (-want +got):\n%s", diff)
			}
		})
	}
}

// layoutRepo creates the given directories and files in an empty repository.
func layoutRepo(t *testing.T, dirs []string, files map[string]string) *afero.Afero {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	for _, d := range dirs {
		err := fs.MkdirAll(filepath.Join(testRepo, d), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	for path, data := range files {
		err := fs.WriteFile(filepath.Join(testRepo, path), []byte(data), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	return fs
}

// clusterBaseVersions maps the path of each adopted layer to its structure
// version, failing on any layer that is not a cluster base.
func clusterBaseVersions(t *testing.T, result *AdoptionResult) map[string]int {
	t.Helper()

	versions := map[string]int{}
	for _, l := range result.Metadata.Layers {
		if l.Kind != LayerClusterBase {
			t.Errorf("unexpected %s layer at %s", l.Kind, l.Path)
		}
		versions[l.Path] = l.StructureVersion
	}

	return versions
}
