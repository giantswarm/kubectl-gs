package migration

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

func Test_NewPlan_MissingMigration(t *testing.T) {
	_, md := testRepository(t)

	_, err := NewPlan(md, nil, 2)
	if !IsMissingMigration(err) {
		t.Fatalf("expected a missing migration error, got: %v", err)
	}
}

func Test_NewPlan_RefusesUnversionedAndNewer(t *testing.T) {
	_, md := testRepository(t)
	md.Layers[0].StructureVersion = 0

	_, err := NewPlan(md, []Migration{renameMigration}, 2)
	if !IsUnversionedLayer(err) {
		t.Errorf("expected an unversioned layer error, got: %v", err)
	}

	_, md = testRepository(t)
	md.Layers[0].StructureVersion = 3

	_, err = NewPlan(md, []Migration{renameMigration}, 2)
	if !IsNewerLayer(err) {
		t.Errorf("expected a newer layer error, got: %v", err)
	}

	// All layers behind, but the repository record ahead: touched by a newer
	// kubectl-gs, which this one must not claim to be up to date with.
	_, md = testRepository(t)
	md.StructureVersion = 2

	_, err = NewPlan(md, []Migration{renameMigration}, 1)
	if !IsNewerLayer(err) {
		t.Errorf("expected a newer repository to be refused, got: %v", err)
	}
}

// A missing repository level version is recovered from the layers when they
// all have one, and refused when there are none to go by.
func Test_NewPlan_UnversionedRepository(t *testing.T) {
	fs, md := testRepository(t)
	md.StructureVersion = 0

	plan, err := NewPlan(md, []Migration{renameMigration}, 1)
	if err != nil {
		t.Fatalf("expected versioned layers to be enough, got: %s", err)
	}
	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if md.StructureVersion != 1 {
		t.Errorf("expected the repository record to be set to 1, got %d", md.StructureVersion)
	}

	_, md = testRepository(t)
	md.StructureVersion = 0
	md.Layers = nil

	_, err = NewPlan(md, []Migration{renameMigration}, 1)
	if !errors.Is(err, unversionedRepositoryError) || strings.Contains(err.Error(), "--adopt") {
		t.Errorf("expected an unversioned repository error without the --adopt advice, got: %v", err)
	}
}

// A duplicate entry recorded at a newer version must not be hidden behind an
// older one when the duplicates are collapsed.
func Test_NewPlan_SeesNewerDuplicates(t *testing.T) {
	_, md := testRepository(t)
	md.Layers = append(md.Layers, metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 3})

	_, err := NewPlan(md, []Migration{renameMigration}, 2)
	if !IsNewerLayer(err) {
		t.Errorf("expected the newer duplicate to be refused, got: %v", err)
	}
	if err != nil && strings.Count(err.Error(), testOrg) != 1 {
		t.Errorf("expected the layer to be listed once, got: %s", err)
	}
}

func Test_NewPlan_CollapsesDuplicateLayers(t *testing.T) {
	fs, md := testRepository(t)
	md.Layers = append(md.Layers, metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 1})

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected a duplicated layer to be migrated once, got %d step(s)", len(plan.Steps))
	}

	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(md.Layers) != 2 {
		t.Errorf("expected the duplicate entry to be collapsed, got %v", md.Layers)
	}
}

func Test_Plan_RunCollapsesDuplicatesAtTarget(t *testing.T) {
	fs, md := testRepository(t)
	md.Layers[0].StructureVersion = 2
	md.Layers = append(md.Layers, md.Layers[0])

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(md.Layers) != 2 {
		t.Errorf("expected duplicates of a layer already at target to be collapsed too, got %v", md.Layers)
	}
}

func Test_Changes_CountsPermissionChanges(t *testing.T) {
	before := map[string]file{"a": {data: []byte("x"), perm: 0600}}
	after := map[string]file{"a": {data: []byte("x"), perm: 0644}}

	got := changes(before, after)
	if len(got) != 1 || got[0].Op != FileModified {
		t.Errorf("expected a permission change to count as modified, got %v", got)
	}
}

func Test_Plan_DryRunWritesNothing(t *testing.T) {
	fs, md := testRepository(t)

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	result, err := plan.Run(fs, testRepo, md, true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantFiles := []FileChange{
		{Op: FileAdded, Path: testOrg + "/NOTES.md"},
		{Op: FileRemoved, Path: testOrg + "/README.md"},
	}
	if fmt.Sprint(result.Files) != fmt.Sprint(wantFiles) {
		t.Errorf("expected files %v, got %v", wantFiles, result.Files)
	}
	if !exists(t, fs, filepath.Join(testRepo, testOrg, "README.md")) || exists(t, fs, filepath.Join(testRepo, testOrg, "NOTES.md")) {
		t.Errorf("expected a dry run to leave the repository alone")
	}
}
