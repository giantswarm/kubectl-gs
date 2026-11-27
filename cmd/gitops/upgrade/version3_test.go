package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/migration"
)

const manualSteps = "Move the Kustomizations by hand."

// manualMigration stands in for a migration that leaves steps to the user,
// and records whether it saw the files git ignores.
func manualMigration(sawIgnored *bool) migration.Migration {
	return migration.Migration{
		From:        1,
		Description: "Organizations get a namespace",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(_ *afero.Afero, _ string, _ metadata.Layer, env migration.Env) ([]string, error) {
			*sawIgnored = env.IsIgnored(testOrg + "/secret.dec.yaml")
			return []string{"added namespace.yaml"}, nil
		},
		ManualSteps: manualSteps,
	}
}

func Test_Upgrade_PrintsManualSteps(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		fs := versionOneRepo(t)
		out := new(bytes.Buffer)

		sawIgnored := false
		r := newTestRunner(fs, 2, out)
		r.migrations = []migration.Migration{manualMigration(&sawIgnored)}
		r.ignoredPaths = func(string) ([]string, error) {
			return []string{testOrg + "/secret.dec.yaml"}, nil
		}

		err := r.run(context.Background(), newTestCommand(testRepo, dryRun), nil)
		if err != nil {
			t.Fatalf("dry run %t: unexpected error: %s", dryRun, err)
		}

		if !strings.Contains(out.String(), manualSteps) {
			t.Errorf("dry run %t: expected the manual steps, got:\n%s", dryRun, out.String())
		}
		if !sawIgnored {
			t.Errorf("dry run %t: expected the migration to see the ignored file", dryRun)
		}
	}
}

// Test_IgnoredPaths checks that the files git ignores are listed relative to
// the repository, as the migrations look them up.
func Test_IgnoredPaths(t *testing.T) {
	dir := isolatedGit(t)
	runGit(t, dir, "init", "-q")
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.dec.yaml\n")
	writeFile(t, filepath.Join(dir, "gitops", "mapi", "secret.dec.yaml"), "kind: Secret\n")
	writeFile(t, filepath.Join(dir, "gitops", "mapi", "kustomization.yaml"), "resources: []\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")

	paths, err := ignoredPaths(filepath.Join(dir, "gitops"))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(paths) != 1 || paths[0] != "mapi/secret.dec.yaml" {
		t.Errorf("expected mapi/secret.dec.yaml, got %v", paths)
	}
}

// Test_Upgrade_FailsWithoutIgnoredFiles checks that the upgrade stops when
// git can't say which files it ignores, rather than moving them.
func Test_Upgrade_FailsWithoutIgnoredFiles(t *testing.T) {
	fs := versionOneRepo(t)
	out := new(bytes.Buffer)

	sawIgnored := false
	r := newTestRunner(fs, 2, out)
	r.migrations = []migration.Migration{manualMigration(&sawIgnored)}
	r.ignoredPaths = func(string) ([]string, error) {
		return nil, errors.New("git ls-files failed")
	}

	err := r.run(context.Background(), newTestCommand(testRepo, false), nil)
	if err == nil || !strings.Contains(err.Error(), "git ls-files failed") {
		t.Fatalf("expected the upgrade to fail, got: %v", err)
	}
}

// Test_Upgrade_RefusedStepWritesNothing checks that the upgrade says
// nothing was written when a migration refuses, as it does so on the
// in-memory copy, before the repository changes, rather than how to undo
// changes that never happened.
func Test_Upgrade_RefusedStepWritesNothing(t *testing.T) {
	refusing := migration.Migration{
		From:        1,
		Description: "Refuses",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(_ *afero.Afero, _ string, _ metadata.Layer, _ migration.Env) ([]string, error) {
			return nil, errors.New("refused")
		},
	}

	for _, tc := range []struct {
		name   string
		dryRun bool
		force  bool
	}{
		{name: "real run"},
		{name: "forced run", force: true},
		{name: "dry run", dryRun: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearRepositoryVars(t)

			r := newTestRunner(versionOneRepo(t).Fs, 2, new(bytes.Buffer))
			r.migrations = []migration.Migration{refusing}
			r.flag.Force = tc.force

			err := r.run(context.Background(), newTestCommand(testRepo, tc.dryRun), nil)
			if err == nil || !strings.Contains(err.Error(), "Nothing was written") || strings.Contains(err.Error(), "git -C") {
				t.Errorf("expected the error to say nothing was written, got: %v", err)
			}
		})
	}
}

// failsOnDisk returns a migration that fails on the file system *onDisk
// points at, and succeeds on any other, such as the in-memory copy a real
// upgrade runs on first.
func failsOnDisk(onDisk **afero.Afero) migration.Migration {
	return migration.Migration{
		From:        1,
		Description: "Fails on disk",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(fs *afero.Afero, _ string, _ metadata.Layer, _ migration.Env) ([]string, error) {
			if fs != *onDisk {
				return nil, nil
			}
			return nil, fmt.Errorf("boom")
		},
	}
}

// failsHalfwayOnDisk returns a migration that goes through on the in-memory
// copy, and fails half way on the repository, after changing a file and
// adding another.
func failsHalfwayOnDisk() migration.Migration {
	return migration.Migration{
		From:        1,
		Description: "Fails half way",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(fs *afero.Afero, repoPath string, layer metadata.Layer, _ migration.Env) ([]string, error) {
			if _, ok := fs.Fs.(*afero.OsFs); !ok {
				return nil, nil
			}
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
}
