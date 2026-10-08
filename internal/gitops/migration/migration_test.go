package migration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	testRepo = "/repo"
	testMC   = "management-clusters/demomc"
	testOrg  = testMC + "/organizations/demoorg"
	testOrg2 = testMC + "/organizations/zorg"
)

// renameMigration stands in for a real structure change: from version 1 to 2,
// organizations get their `README.md` renamed to `NOTES.md`.
var renameMigration = Migration{
	From:        1,
	Description: "Rename organization README.md to NOTES.md",
	Kinds:       []string{metadata.LayerOrganization},
	Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
		from := filepath.Join(repoPath, layer.Path, "README.md")
		to := filepath.Join(repoPath, layer.Path, "NOTES.md")

		err := fs.Rename(from, to)
		if err != nil {
			return nil, err
		}

		return []string{fmt.Sprintf("renamed %s to NOTES.md", filepath.Join(layer.Path, "README.md"))}, nil
	},
}

// recording records the steps its migrations are applied in, and the layer
// version each one saw.
type recording struct {
	steps []string
}

func (r *recording) migration(from int, kinds ...string) Migration {
	return Migration{
		From:        from,
		Description: fmt.Sprintf("step from %d", from),
		Kinds:       kinds,
		Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
			r.steps = append(r.steps, fmt.Sprintf("%d:%s@%d", from, layer.Path, layer.StructureVersion))
			return nil, nil
		},
	}
}

func testRepository(t *testing.T) (*afero.Afero, *metadata.RepositoryMetadata) {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	for _, org := range []string{testOrg, testOrg2} {
		err := fs.WriteFile(filepath.Join(testRepo, org, "README.md"), []byte("hello"), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	md := metadata.New()
	md.StructureVersion = 1
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 1, GeneratedWith: "kubectl-gs/old"},
		{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 1, GeneratedWith: "kubectl-gs/old"},
	}

	return fs, md
}

func exists(t *testing.T, fs *afero.Afero, path string) bool {
	t.Helper()

	ok, err := fs.Exists(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	return ok
}

func Test_Registry_CoversEveryStructureVersion(t *testing.T) {
	seen := map[int]bool{}
	for _, m := range Migrations {
		if seen[m.From] {
			t.Errorf("more than one migration registered from structure version %d", m.From)
		}
		seen[m.From] = true

		if m.From < 1 || m.From >= metadata.StructureVersion {
			t.Errorf("migration from structure version %d is outside 1..%d", m.From, metadata.StructureVersion-1)
		}
		if m.Description == "" || m.Apply == nil || len(m.Kinds) == 0 {
			t.Errorf("migration from structure version %d needs a description, the kinds it affects and an Apply", m.From)
		}
	}

	for v := 1; v < metadata.StructureVersion; v++ {
		if !seen[v] {
			t.Errorf("metadata.StructureVersion is %d but no migration is registered from version %d", metadata.StructureVersion, v)
		}
	}
}

func Test_NewPlan_NothingToDoAtTarget(t *testing.T) {
	_, md := testRepository(t)

	plan, err := NewPlan(md, []Migration{renameMigration}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !plan.Empty() {
		t.Errorf("expected an empty plan, got %d step(s) over %d layer(s)", len(plan.Steps), len(plan.Layers))
	}
}

func Test_NewPlan_OnlyAffectedKindsGetSteps(t *testing.T) {
	_, md := testRepository(t)

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(plan.Layers) != 2 {
		t.Errorf("expected both layers to be bumped, got %d", len(plan.Layers))
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Layer.Path != testOrg {
		t.Errorf("expected a single step for the organization, got %v", plan.Steps)
	}
}

func Test_Plan_OrdersVersionFirstAndPassesEachStepItsVersion(t *testing.T) {
	fs, md := testRepository(t)
	md.Layers = append(md.Layers, metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg2, StructureVersion: 2})

	rec := &recording{}
	migrations := []Migration{
		rec.migration(2, metadata.LayerOrganization, metadata.LayerManagementCluster),
		rec.migration(1, metadata.LayerOrganization, metadata.LayerManagementCluster),
	}

	plan, err := NewPlan(md, migrations, 3)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := []string{
		"1:" + testMC + "@1",
		"1:" + testOrg + "@1",
		"2:" + testMC + "@2",
		"2:" + testOrg + "@2",
		"2:" + testOrg2 + "@2",
	}
	if fmt.Sprint(rec.steps) != fmt.Sprint(want) {
		t.Errorf("expected steps\n  %v\ngot\n  %v", want, rec.steps)
	}

	for _, l := range md.Layers {
		if l.StructureVersion != 3 {
			t.Errorf("expected %s at version 3, got %d", l.Path, l.StructureVersion)
		}
	}
}

func Test_NewPlan_MissingMigration(t *testing.T) {
	_, md := testRepository(t)

	_, err := NewPlan(md, nil, 2)
	if err == nil {
		t.Fatalf("expected an error for a missing migration")
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
	if microerror.Cause(err) != unversionedRepositoryError || strings.Contains(err.Error(), "--adopt") {
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

func Test_Plan_Run(t *testing.T) {
	fs, md := testRepository(t)

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	result, err := plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(result.Changes) != 1 || len(result.Changes[0].Changes) != 1 {
		t.Errorf("expected one reported change, got %v", result.Changes)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected a real run to leave listing files to git, got %v", result.Files)
	}

	if !exists(t, fs, filepath.Join(testRepo, testOrg, "NOTES.md")) || exists(t, fs, filepath.Join(testRepo, testOrg, "README.md")) {
		t.Errorf("expected the migration to have renamed README.md on disk")
	}

	if md.StructureVersion != 2 {
		t.Errorf("expected the repository to be at version 2, got %d", md.StructureVersion)
	}
	for _, l := range md.Layers {
		if l.StructureVersion != 2 {
			t.Errorf("expected %s to be at version 2, got %d", l.Path, l.StructureVersion)
		}
		if l.GeneratedWith != metadata.GeneratedWith() {
			t.Errorf("expected %s to be stamped with %s, got %s", l.Path, metadata.GeneratedWith(), l.GeneratedWith)
		}
	}
}

func Test_Plan_RunRecordsRepositoryLevelOnly(t *testing.T) {
	fs, md := testRepository(t)
	md.Layers = nil
	md.StructureVersion = 1

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !plan.Empty() {
		t.Fatalf("expected no layer to need upgrading")
	}

	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if md.StructureVersion != 2 {
		t.Errorf("expected the repository record to move to 2, got %d", md.StructureVersion)
	}
}

// A migration failing on the second layer leaves the first one migrated on
// disk, which is what git is there to undo, but must not record anything.
func Test_Plan_RunFailureRecordsNothing(t *testing.T) {
	fs, md := testRepository(t)
	md.Layers = append(md.Layers, metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg2, StructureVersion: 1})
	_ = fs.Remove(filepath.Join(testRepo, testOrg2, "README.md"))

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for _, dryRun := range []bool{true, false} {
		_, err = plan.Run(fs, testRepo, md, dryRun)
		if !IsMigrationFailed(err) {
			t.Fatalf("dry run %t: expected a failed migration, got: %v", dryRun, err)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("dry run %t: expected the migration's own error to be kept, got: %v", dryRun, err)
		}

		if dryRun && !exists(t, fs, filepath.Join(testRepo, testOrg, "README.md")) {
			t.Errorf("expected a failed dry run to leave the repository alone")
		}
	}

	for _, l := range md.Layers {
		if l.StructureVersion != 1 {
			t.Errorf("expected %s to stay at version 1 after a failed run, got %d", l.Path, l.StructureVersion)
		}
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

// Against a real clone: `.git`, symlinks and a relative repository path must
// neither break the run nor be touched by it.
func Test_Plan_RunOnDisk(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	fs := &afero.Afero{Fs: afero.NewOsFs()}
	for path, data := range map[string]string{
		".git/HEAD":                      "ref: refs/heads/main",
		testOrg + "/README.md":           "hello",
		testMC + "/secrets/key.enc.yaml": "secret",
		testOrg + "/submodule/.git":      "gitdir: ../../.git/modules/submodule",
	} {
		err := fs.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		err = fs.WriteFile(path, []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	for link, target := range map[string]string{
		testMC + "/linked-dir": "secrets",
		testMC + "/dangling":   "does-not-exist",
	} {
		err := os.Symlink(target, link)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	md := metadata.New()
	md.StructureVersion = 1
	md.Layers = []metadata.Layer{{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 1}}

	plan, err := NewPlan(md, []Migration{renameMigration}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	result, err := plan.Run(fs, ".", md, true)
	if err != nil {
		t.Fatalf("dry run: unexpected error: %s", err)
	}
	for _, f := range result.Files {
		if f.Path != testOrg+"/NOTES.md" && f.Path != testOrg+"/README.md" {
			t.Errorf("expected the dry run to report only the renamed file, got %v", result.Files)
		}
	}
	if !exists(t, fs, testOrg+"/README.md") {
		t.Fatalf("expected the dry run to leave README.md on disk")
	}

	_, err = plan.Run(fs, ".", md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !exists(t, fs, testOrg+"/NOTES.md") || exists(t, fs, testOrg+"/README.md") {
		t.Errorf("expected README.md to be renamed on disk")
	}
	for _, link := range []string{testMC + "/linked-dir", testMC + "/dangling"} {
		if _, err := os.Lstat(link); err != nil {
			t.Errorf("expected symlink %s to be left alone, got: %s", link, err)
		}
	}
	if !exists(t, fs, ".git/HEAD") {
		t.Errorf("expected .git to be left alone")
	}
}
