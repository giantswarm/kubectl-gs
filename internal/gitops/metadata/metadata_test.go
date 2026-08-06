package metadata

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

func newTestFs() *afero.Afero {
	return &afero.Afero{Fs: afero.NewMemMapFs()}
}

func Test_Load_NotFound(t *testing.T) {
	fs := newTestFs()

	_, err := Load(fs, "/repo")
	if !IsNotFound(err) {
		t.Fatalf("expected a NotFoundError, got: %v", err)
	}
}

func Test_LoadOrNew_FallsBackToEmpty(t *testing.T) {
	fs := newTestFs()

	md, err := LoadOrNew(fs, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if md.StructureVersion != StructureVersion {
		t.Errorf("expected structure version %d, got %d", StructureVersion, md.StructureVersion)
	}
	if len(md.Layers) != 0 {
		t.Errorf("expected no layers, got %d", len(md.Layers))
	}
}

func Test_SaveLoad_RoundTrip(t *testing.T) {
	fs := newTestFs()

	md := New()
	md.Upsert(Layer{Kind: LayerManagementCluster, Path: "management-clusters/demomc"})
	md.Upsert(Layer{Kind: LayerOrganization, Path: "management-clusters/demomc/organizations/demoorg"})

	err := Save(fs, "/repo", md)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	loaded, err := Load(fs, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if diff := cmp.Diff(md, loaded); diff != "" {
		t.Errorf("metadata did not survive the round trip (-want +got):\n%s", diff)
	}
}

func Test_Upsert_IsIdempotent(t *testing.T) {
	md := New()

	md.Upsert(Layer{Kind: LayerWorkloadCluster, Path: "a/b/c"})
	md.Upsert(Layer{Kind: LayerWorkloadCluster, Path: "a/b/c"})

	if len(md.Layers) != 1 {
		t.Fatalf("expected re-adding the same layer to update it in place, got %d entries", len(md.Layers))
	}
}

func Test_Upsert_DistinguishesKinds(t *testing.T) {
	md := New()

	// A path could in principle be recorded under two kinds; they must not
	// collapse into one entry.
	md.Upsert(Layer{Kind: LayerWorkloadCluster, Path: "a/b/c"})
	md.Upsert(Layer{Kind: LayerApp, Path: "a/b/c"})

	if len(md.Layers) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(md.Layers))
	}
}

func Test_Upsert_RefreshesVersion(t *testing.T) {
	md := New()

	md.Upsert(Layer{Kind: LayerOrganization, Path: "a/b", StructureVersion: 0, GeneratedWith: "kubectl-gs/0.0.1"})
	// A recorded 0 means "unset" and is filled in with the current version.
	if got := md.Layers[0].StructureVersion; got != StructureVersion {
		t.Errorf("expected an unset version to default to %d, got %d", StructureVersion, got)
	}

	md.Upsert(Layer{Kind: LayerOrganization, Path: "a/b"})
	if got := md.Layers[0].GeneratedWith; got != GeneratedWith() {
		t.Errorf("expected re-adding to refresh generatedWith to %q, got %q", GeneratedWith(), got)
	}
}

func Test_MergeMissing_LeavesRecordedLayersAlone(t *testing.T) {
	md := New()
	md.Upsert(Layer{Kind: LayerOrganization, Path: "a/b", StructureVersion: 1})
	md.Layers[0].StructureVersion = 0 // pretend it was recorded by an older kubectl-gs

	added := md.MergeMissing([]Layer{
		{Kind: LayerOrganization, Path: "a/b"},
		{Kind: LayerWorkloadCluster, Path: "a/b/c"},
	})

	if len(added) != 1 || added[0].Path != "a/b/c" {
		t.Fatalf("expected only the untracked layer to be added, got %v", added)
	}
	if md.Layers[0].StructureVersion != 0 {
		t.Errorf("merging must not bump an already recorded layer, got version %d", md.Layers[0].StructureVersion)
	}
}

func Test_Stamp_NeverMovesBackwards(t *testing.T) {
	md := New()
	md.StructureVersion = StructureVersion + 5

	md.Stamp()

	if md.StructureVersion != StructureVersion+5 {
		t.Errorf("a repository touched by a newer kubectl-gs must not be downgraded, got %d", md.StructureVersion)
	}
}

func Test_OutdatedLayers(t *testing.T) {
	md := New()
	md.Upsert(Layer{Kind: LayerOrganization, Path: "current"})
	md.Upsert(Layer{Kind: LayerOrganization, Path: "old", StructureVersion: StructureVersion})
	md.Layers[1].StructureVersion = StructureVersion - 1

	outdated := md.OutdatedLayers()
	if len(outdated) != 1 || outdated[0].Path != "old" {
		t.Fatalf("expected only the older layer to be reported, got %v", outdated)
	}
}

func Test_Load_RejectsForeignFile(t *testing.T) {
	fs := newTestFs()

	err := fs.WriteFile(FilePath("/repo"), []byte("apiVersion: v1\nkind: ConfigMap\n"), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_, err = Load(fs, "/repo")
	if err == nil {
		t.Fatal("expected loading a file of another kind to fail")
	}
	if IsNotFound(err) {
		t.Fatal("a file of the wrong kind must not be reported as missing, or it would be silently replaced")
	}
}

func Test_Render_IsStableAndCommented(t *testing.T) {
	first := New()
	first.Upsert(Layer{Kind: LayerWorkloadCluster, Path: "z/last"})
	first.Upsert(Layer{Kind: LayerManagementCluster, Path: "a/first"})

	second := New()
	second.Upsert(Layer{Kind: LayerManagementCluster, Path: "a/first"})
	second.Upsert(Layer{Kind: LayerWorkloadCluster, Path: "z/last"})

	firstBody, err := Render(first)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	secondBody, err := Render(second)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if diff := cmp.Diff(string(firstBody), string(secondBody)); diff != "" {
		t.Errorf("insertion order must not change the rendered file (-first +second):\n%s", diff)
	}

	if got := string(firstBody[0]); got != "#" {
		t.Errorf("expected the rendered file to start with the do-not-edit comment, got %q", got)
	}
}
