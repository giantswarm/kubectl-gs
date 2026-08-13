package metadata

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

const testRepo = "/repo"

// fixtureRepo lays out a repository the way `kubectl gs gitops` would have,
// covering both the `mapi` and the `--skip-mapi` variants, an app created from
// a base (Kustomization, no App CR), a cluster base, and two directories that
// are not layers at all.
func fixtureRepo(t *testing.T) *afero.Afero {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}

	dirs := []string{
		"bases/clusters/capa/template",
		"management-clusters/demomc/secrets",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/ingress-from-base",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/cluster",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/flatwc/apps/hello-world",
		"management-clusters/demomc/organizations/otherorg/workload-clusters/otherwc/mapi/apps",
		// Not a layer: an app directory holding neither an App CR nor a
		// Kustomization.
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/leftovers",
		// Not a layer: a provider directory without the `template` subdir.
		"bases/clusters/capz",
	}
	for _, d := range dirs {
		err := fs.MkdirAll(filepath.Join(testRepo, d), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	files := []string{
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world/appcr.yaml",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/ingress-from-base/kustomization.yaml",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/flatwc/apps/hello-world/appcr.yaml",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/leftovers/notes.txt",
	}
	for _, f := range files {
		err := fs.WriteFile(filepath.Join(testRepo, f), []byte("---\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	return fs
}

func Test_Adopt(t *testing.T) {
	fs := fixtureRepo(t)

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expected := []Layer{
		{Kind: LayerClusterBase, Path: "bases/clusters/capa/template"},
		{Kind: LayerManagementCluster, Path: "management-clusters/demomc"},
		{Kind: LayerOrganization, Path: "management-clusters/demomc/organizations/demoorg"},
		{Kind: LayerWorkloadCluster, Path: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc"},
		{Kind: LayerApp, Path: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world"},
		{Kind: LayerApp, Path: "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/ingress-from-base"},
		{Kind: LayerWorkloadCluster, Path: "management-clusters/demomc/organizations/demoorg/workload-clusters/flatwc"},
		{Kind: LayerApp, Path: "management-clusters/demomc/organizations/demoorg/workload-clusters/flatwc/apps/hello-world"},
		{Kind: LayerOrganization, Path: "management-clusters/demomc/organizations/otherorg"},
		{Kind: LayerWorkloadCluster, Path: "management-clusters/demomc/organizations/otherorg/workload-clusters/otherwc"},
	}

	got := make([]Layer, 0, len(result.Metadata.Layers))
	for _, l := range result.Metadata.Layers {
		got = append(got, Layer{Kind: l.Kind, Path: l.Path})
	}

	want := &RepositoryMetadata{Layers: expected}
	want.Sort()

	if diff := cmp.Diff(want.Layers, got); diff != "" {
		t.Errorf("adopted layers differ (-want +got):\n%s", diff)
	}
}

func Test_Adopt_ReportsWhatItSkipped(t *testing.T) {
	fs := fixtureRepo(t)

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expected := []string{
		"bases/clusters/capz",
		"management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/leftovers",
	}

	if diff := cmp.Diff(expected, result.Skipped); diff != "" {
		t.Errorf("skipped directories differ (-want +got):\n%s", diff)
	}
}

func Test_Adopt_StampsCurrentVersion(t *testing.T) {
	fs := fixtureRepo(t)

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for _, l := range result.Metadata.Layers {
		if l.StructureVersion != StructureVersion {
			t.Errorf("layer %s recorded at version %d, expected %d", l.Path, l.StructureVersion, StructureVersion)
		}
	}
}

func Test_Adopt_EmptyRepository(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("walking a repository that does not exist must not fail: %s", err)
	}

	if len(result.Metadata.Layers) != 0 {
		t.Errorf("expected no layers, got %d", len(result.Metadata.Layers))
	}
}
