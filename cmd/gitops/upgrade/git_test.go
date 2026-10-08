package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/migration"
)

// isolatedGit makes sure git is installed, and that the repositories a test
// creates are not affected by the environment it runs in: no GIT_* variables,
// no enclosing repository, no user or system configuration.
func isolatedGit(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "GIT_") {
			t.Setenv(name, "")
			_ = os.Unsetenv(name)
		}
	}

	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	return dir
}

// clearRepositoryVars unsets the repositoryVars for the test, e.g. the
// GIT_INDEX_FILE git sets when the tests run from a pre-commit hook, and
// restores them afterwards.
func clearRepositoryVars(t *testing.T) {
	t.Helper()

	for _, name := range repositoryVars {
		t.Setenv(name, "") // restores the variable once the test is done
		_ = os.Unsetenv(name)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %s: %s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	err = os.WriteFile(path, []byte(data), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

// metadataRepo lays out a version one gitops repository at root/path, without
// touching git.
func metadataRepo(t *testing.T, root, path string) string {
	t.Helper()

	repo := filepath.Join(root, path)
	writeFile(t, filepath.Join(repo, testOrg, "kustomization.yaml"), "resources: []\n")

	md := metadata.New()
	md.StructureVersion = 1
	md.Upsert(metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 1})
	err := metadata.Save(&afero.Afero{Fs: afero.NewOsFs()}, repo, md)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	return repo
}

func Test_DirtyPaths(t *testing.T) {
	dir := isolatedGit(t)

	_, err := dirtyPaths(dir)
	if !errors.Is(err, errNotTracked) {
		t.Errorf("expected a directory outside git to be reported as not tracked, got: %v", err)
	}

	runGit(t, dir, "init", "-q")
	_, err = dirtyPaths(dir)
	if !errors.Is(err, errNotTracked) {
		t.Errorf("expected a repository with nothing committed to be reported as not tracked, got: %v", err)
	}

	writeFile(t, filepath.Join(dir, "kustomization.yaml"), "x")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	dirty, err := dirtyPaths(dir)
	if err != nil || len(dirty) != 0 {
		t.Errorf("expected a committed repository to be clean, got %v, %v", dirty, err)
	}

	writeFile(t, filepath.Join(dir, "untracked.yaml"), "x")
	dirty, err = dirtyPaths(dir)
	if err != nil || len(dirty) != 1 {
		t.Errorf("expected an untracked file to make the tree dirty, got %v, %v", dirty, err)
	}
}

// A repository ignored by the one it sits in reports a clean status whatever
// its state, and the restore command cannot restore it: it must be refused.
func Test_DirtyPaths_IgnoredDirectory(t *testing.T) {
	dir := isolatedGit(t)
	runGit(t, dir, "init", "-q")
	writeFile(t, filepath.Join(dir, ".gitignore"), "gitops/\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	repo := metadataRepo(t, dir, "gitops")

	_, err := dirtyPaths(repo)
	if !errors.Is(err, errNotTracked) {
		t.Errorf("expected an ignored directory to be reported as not tracked, got: %v", err)
	}
}

// A repository nested in a larger one is checked on its own: changes elsewhere
// in the larger one neither hold it up nor show in the error.
func Test_DirtyPaths_ScopedToRepository(t *testing.T) {
	dir := isolatedGit(t)
	runGit(t, dir, "init", "-q")
	repo := metadataRepo(t, dir, "gitops")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	writeFile(t, filepath.Join(dir, "outside.txt"), "x")

	dirty, err := dirtyPaths(repo)
	if err != nil || len(dirty) != 0 {
		t.Errorf("expected changes outside the repository to be ignored, got %v, %v", dirty, err)
	}

	writeFile(t, filepath.Join(repo, "inside.txt"), "x")

	dirty, err = dirtyPaths(repo)
	if err != nil || len(dirty) != 1 || !strings.Contains(dirty[0], "inside.txt") {
		t.Errorf("expected only the change inside the repository, got %v, %v", dirty, err)
	}
}

// End to end against real git: a migration failing part way leaves a change
// on disk, and the restore command the error prints undoes it.
func Test_Upgrade_RestoreHintUndoesAFailedRun(t *testing.T) {
	dir := isolatedGit(t)
	repo := metadataRepo(t, dir, "")
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-q", "-m", "init")

	halfway := migration.Migration{
		From:        1,
		Description: "Fails half way",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
			err := fs.WriteFile(filepath.Join(repoPath, layer.Path, "kustomization.yaml"), []byte("changed\n"), 0600)
			if err != nil {
				return nil, err
			}
			err = fs.WriteFile(filepath.Join(repoPath, layer.Path, "new.yaml"), []byte("new\n"), 0600)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("boom")
		},
	}

	out := new(bytes.Buffer)
	r := &runner{
		dirtyPaths: dirtyPaths,
		flag:       &flag{},
		fs:         &afero.Afero{Fs: afero.NewOsFs()},
		migrations: []migration.Migration{halfway},
		stdout:     out,
		stderr:     out,
		target:     2,
	}

	err := r.run(context.Background(), newTestCommand(repo, false), nil)
	if microerror.Cause(err) != upgradeFailedError {
		t.Fatalf("expected the upgrade to fail, got: %v", err)
	}

	hint := restoreHint(repo)
	if !strings.Contains(err.Error(), hint) {
		t.Fatalf("expected the error to print %q, got: %s", hint, err)
	}

	out2, err := exec.Command("sh", "-c", hint).CombinedOutput()
	if err != nil {
		t.Fatalf("running the restore command: %s: %s", err, out2)
	}

	dirty, err := dirtyPaths(repo)
	if err != nil || len(dirty) != 0 {
		t.Errorf("expected the restore command to leave a clean tree, got %v, %v", dirty, err)
	}
}

func Test_GitEnv(t *testing.T) {
	env := gitEnv([]string{"PATH=/bin", "GIT_DIR=/other/.git", "GIT_WORK_TREE=/other", "LANG=de_DE.UTF-8", "LC_ALL=de_DE.UTF-8", "GIT_CONFIG_GLOBAL=/dev/null"})

	want := "[PATH=/bin GIT_CONFIG_GLOBAL=/dev/null LC_ALL=C]"
	if got := fmt.Sprint(env); got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func Test_RestoreHint_UnsetsGitDir(t *testing.T) {
	clearRepositoryVars(t)
	if hint := restoreHint("/repo"); strings.Contains(hint, "env ") {
		t.Errorf("expected no env prefix without GIT_DIR, got: %s", hint)
	}

	t.Setenv("GIT_DIR", "/other/.git")
	t.Setenv("GIT_COMMON_DIR", "/other/.git")
	want := "env -u GIT_DIR -u GIT_COMMON_DIR git -C /repo checkout -- . && env -u GIT_DIR -u GIT_COMMON_DIR git -C /repo clean -fd"
	if hint := restoreHint("/repo"); hint != want {
		t.Errorf("expected %q, got %q", want, hint)
	}

	// Every git command suggested gets the same prefix, not only the restore.
	if got := gitCommand("/repo", "diff"); got != "env -u GIT_DIR -u GIT_COMMON_DIR git -C /repo diff" {
		t.Errorf("expected the diff command to unset the same variables, got %q", got)
	}
}

// Detecting a directory outside git must not depend on the user's locale.
func Test_DirtyPaths_IgnoresLocale(t *testing.T) {
	dir := isolatedGit(t)
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	t.Setenv("LANG", "de_DE.UTF-8")

	_, err := dirtyPaths(dir)
	if !errors.Is(err, errNotTracked) {
		t.Errorf("expected a directory outside git to be reported as not tracked, got: %v", err)
	}
}
