package upgrade

import (
	"bytes"
	"context"
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
		Apply: func(fs *afero.Afero, _ string, _ metadata.Layer) ([]string, error) {
			*sawIgnored = migration.EnvOf(fs).IsIgnored(testOrg + "/secret.dec.yaml")
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
