package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/migration"
)

const (
	testRepo = "/repo"
	testMC   = "management-clusters/demomc"
	testOrg  = testMC + "/organizations/demoorg"
)

// addNotesMigration stands in for a real structure change: from version 1 to
// 2, organizations get a NOTES.md.
var addNotesMigration = migration.Migration{
	From:        1,
	Description: "Organizations get a NOTES.md",
	Kinds:       []string{metadata.LayerOrganization},
	Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
		err := fs.WriteFile(filepath.Join(repoPath, layer.Path, "NOTES.md"), []byte("notes"), 0600)
		if err != nil {
			return nil, err
		}

		return []string{"added NOTES.md"}, nil
	},
}

// newTestCommand mimics the cobra tree the runner reads its inherited flags
// from: `gitops` owns --local-path and --dry-run, `upgrade` is its child.
func newTestCommand(repoPath string, dryRun bool) *cobra.Command {
	parent := &cobra.Command{Use: "gitops"}
	parent.PersistentFlags().String("local-path", ".", "")
	parent.PersistentFlags().Bool("dry-run", false, "")
	_ = parent.PersistentFlags().Set("local-path", repoPath)
	if dryRun {
		_ = parent.PersistentFlags().Set("dry-run", "true")
	}

	child := &cobra.Command{Use: "upgrade"}
	parent.AddCommand(child)

	return child
}

func cleanTree(string) ([]string, error) { return nil, nil }

func newTestRunner(fs afero.Fs, target int, out *bytes.Buffer) *runner {
	return &runner{
		dirtyPaths: cleanTree,
		flag:       &flag{},
		fs:         &afero.Afero{Fs: fs},
		migrations: []migration.Migration{addNotesMigration},
		stdout:     out,
		stderr:     out,
		target:     target,
	}
}

// versionOneRepo lays out a repository recorded at structure version 1.
func versionOneRepo(t *testing.T) *afero.Afero {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	err := fs.MkdirAll(filepath.Join(testRepo, testOrg), 0755)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	md := metadata.New()
	md.StructureVersion = 1
	md.Upsert(metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 1})
	md.Upsert(metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 1})
	err = metadata.Save(fs, testRepo, md)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	return fs
}

func Test_Upgrade_NothingToDo(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	err := newTestRunner(fs.Fs, 1, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !strings.Contains(out.String(), "nothing to upgrade") {
		t.Errorf("expected the command to say there is nothing to upgrade, got:\n%s", out.String())
	}
}

func Test_Upgrade_FailsWithoutMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	out := new(bytes.Buffer)
	_ = fs.MkdirAll(testRepo, 0755)

	err := newTestRunner(fs, 1, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err == nil || !strings.Contains(err.Error(), "--adopt") {
		t.Errorf("expected an error pointing at --adopt, got: %v", err)
	}
}

func Test_Upgrade_MigratesAndRecords(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	err := newTestRunner(fs.Fs, 2, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	exists, _ := fs.Exists(filepath.Join(testRepo, testOrg, "NOTES.md"))
	if !exists {
		t.Errorf("expected the migration to have added NOTES.md")
	}

	md, err := metadata.Load(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if md.StructureVersion != 2 {
		t.Errorf("expected the repository to be recorded at version 2, got %d", md.StructureVersion)
	}
	for _, l := range md.Layers {
		if l.StructureVersion != 2 {
			t.Errorf("expected %s to be recorded at version 2, got %d", l.Path, l.StructureVersion)
		}
	}

	for _, want := range []string{
		"Upgraded 2 layer(s)",
		"1 -> 2: Organizations get a NOTES.md",
		"added NOTES.md",
		"only their recorded version moves:\n  management-cluster " + testMC,
		"Wrote ",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("expected the output to contain %q, got:\n%s", want, out.String())
		}
	}
}

func Test_Upgrade_DryRunWritesNothing(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	before, _ := fs.ReadFile(metadata.FilePath(testRepo))

	err := newTestRunner(fs.Fs, 2, out).run(context.Background(), newTestCommand(testRepo, true), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	exists, _ := fs.Exists(filepath.Join(testRepo, testOrg, "NOTES.md"))
	if exists {
		t.Errorf("expected a dry run not to add NOTES.md")
	}

	after, _ := fs.ReadFile(metadata.FilePath(testRepo))
	if !bytes.Equal(before, after) {
		t.Errorf("expected a dry run to leave %s alone", metadata.FileName)
	}

	if !strings.Contains(out.String(), "Would upgrade 2 layer(s)") || !strings.Contains(out.String(), "added  "+testOrg+"/NOTES.md") || !strings.Contains(out.String(), "structureVersion: 2") {
		t.Errorf("expected the dry run to report the upgrade and the resulting metadata, got:\n%s", out.String())
	}
}

func Test_Upgrade_RefusesDirtyTree(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	r := newTestRunner(fs.Fs, 2, out)
	r.dirtyPaths = func(string) ([]string, error) { return []string{" M secrets.yaml"}, nil }

	err := r.run(context.Background(), newTestCommand(testRepo, false), nil)
	if microerror.Cause(err) != dirtyTreeError {
		t.Fatalf("expected a dirty tree to be refused, got: %v", err)
	}
	if !strings.Contains(err.Error(), "secrets.yaml") || !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected the error to name the changes and --force, got: %s", err)
	}
	if exists, _ := fs.Exists(filepath.Join(testRepo, testOrg, "NOTES.md")); exists {
		t.Errorf("expected nothing to be migrated")
	}

	// A dry run writes nothing, so it has nothing to protect.
	out.Reset()
	err = r.run(context.Background(), newTestCommand(testRepo, true), nil)
	if err != nil {
		t.Errorf("expected a dry run to ignore a dirty tree, got: %s", err)
	}

	// --force goes ahead anyway.
	r.flag.Force = true
	err = r.run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("expected --force to upgrade a dirty tree, got: %s", err)
	}
	if exists, _ := fs.Exists(filepath.Join(testRepo, testOrg, "NOTES.md")); !exists {
		t.Errorf("expected --force to migrate")
	}
}

func Test_Upgrade_RefusesWithoutGit(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{err: errNotTracked, want: "git tracks nothing under /repo"},
		{err: errGitNotFound, want: "git is not installed"},
	} {
		fs := versionOneRepo(t)
		out := new(bytes.Buffer)

		r := newTestRunner(fs.Fs, 2, out)
		r.dirtyPaths = func(string) ([]string, error) { return nil, tc.err }

		err := r.run(context.Background(), newTestCommand(testRepo, false), nil)
		if microerror.Cause(err) != gitError || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "--force") {
			t.Errorf("expected %q and --force, got: %v", tc.want, err)
		}
		if strings.Count(err.Error(), "git error") > 1 {
			t.Errorf("expected the error kind once, got: %s", err)
		}
	}
}

func Test_Upgrade_RefusesMissingPath(t *testing.T) {
	out := new(bytes.Buffer)

	err := newTestRunner(afero.NewMemMapFs(), 2, out).run(context.Background(), newTestCommand("/does-not-exist", false), nil)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected a missing --local-path to be reported as such, got: %v", err)
	}
}

func Test_Upgrade_FailureHints(t *testing.T) {
	failing := migration.Migration{
		From:        1,
		Description: "Always fails",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
			return nil, fmt.Errorf("boom")
		},
	}

	for _, tc := range []struct {
		name      string
		dryRun    bool
		force     bool
		untracked bool
		want      string
	}{
		{name: "real run", want: "git -C /repo checkout -- . && git -C /repo clean -fd"},
		{name: "forced run", force: true, want: "git -C /repo status"},
		{name: "forced run outside git", force: true, untracked: true, want: "cannot be restored automatically"},
		{name: "dry run", dryRun: true, want: "Nothing was written"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearRepositoryVars(t)

			fs := versionOneRepo(t)
			out := new(bytes.Buffer)

			r := newTestRunner(fs.Fs, 2, out)
			r.migrations = []migration.Migration{failing}
			r.flag.Force = tc.force
			if tc.untracked {
				r.dirtyPaths = func(string) ([]string, error) { return nil, errNotTracked }
			}

			err := r.run(context.Background(), newTestCommand(testRepo, tc.dryRun), nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected the error to contain %q, got: %v", tc.want, err)
			}
			if tc.force && strings.Contains(err.Error(), "clean -fd") {
				t.Errorf("expected --force not to suggest resetting the tree, got: %s", err)
			}
		})
	}
}

func Test_ShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"/repo":            "/repo",
		"/my repo":         "'/my repo'",
		"/it's":            `'/it'\''s'`,
		"":                 "''",
		"./a/b-c_d.e/f@g=": "./a/b-c_d.e/f@g=",
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

// Outside git, a successful upgrade must not suggest a git command that fails.
func Test_Upgrade_OutsideGitSuggestsNoGitDiff(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	r := newTestRunner(fs.Fs, 2, out)
	r.flag.Force = true
	r.dirtyPaths = func(string) ([]string, error) { return nil, errNotTracked }

	err := r.run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if strings.Contains(out.String(), "git -C") {
		t.Errorf("expected no git command to be suggested outside git, got:\n%s", out.String())
	}
}

// A version only bump runs no migration, so it is not guarded, but outside git
// it must still not suggest a git command.
func Test_Upgrade_VersionOnlyOutsideGit(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	r := newTestRunner(fs.Fs, 2, out)
	r.migrations = []migration.Migration{{
		From:        1,
		Description: "Affects no recorded layer",
		Kinds:       []string{metadata.LayerClusterBase},
		Apply: func(*afero.Afero, string, metadata.Layer) ([]string, error) {
			return nil, fmt.Errorf("must not run")
		},
	}}
	r.dirtyPaths = func(string) ([]string, error) { return []string{" M dirty.yaml"}, errNotTracked }

	err := r.run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("expected a version only bump to go ahead, got: %s", err)
	}
	if strings.Contains(out.String(), "git -C") {
		t.Errorf("expected no git command to be suggested outside git, got:\n%s", out.String())
	}

	md, err := metadata.Load(fs, testRepo)
	if err != nil || md.StructureVersion != 2 {
		t.Errorf("expected the repository to be recorded at 2, got %v, %v", md, err)
	}
}
