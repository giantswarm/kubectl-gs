package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// v1WorkloadCluster holds the files of a workload cluster as kubectl-gs
// generated them at structure version 1, with `--skip-mapi`.
func v1WorkloadCluster() map[string]string {
	return map[string]string{
		testWC + "/cluster/kustomization.yaml":            v1ClusterKustomization,
		testWC + "/cluster/patch_cluster_userconfig.yaml": v1ClusterUserConfigPatch,
		testWC + "/cluster/cluster_userconfig.yaml":       "apiVersion: v1\nkind: ConfigMap\n",
		testWC + "/apps/kustomization.yaml":               v1AppsKustomization,
		testWC + "/apps/patch_cluster_config.yaml":        v1ClusterConfigPatch,
	}
}

// Test_helmReleases_LeavesIgnoredFiles checks that the migration never
// changes a file git ignores, which git cannot restore. A patch it would
// rewrite, or a kustomization.yaml it would edit, is refused, so the user
// rewrites it by hand; a shared patch it would remove is left, and reported.
func Test_helmReleases_LeavesIgnoredFiles(t *testing.T) {
	testCases := []struct {
		name        string
		ignored     string
		expectErr   bool
		expectFiles map[string]string
	}{
		{name: "user config patch", ignored: testWC + "/cluster/patch_cluster_userconfig.yaml", expectErr: true},
		{name: "cluster kustomization.yaml", ignored: testWC + "/cluster/kustomization.yaml", expectErr: true},
		{name: "apps kustomization.yaml", ignored: testWC + "/apps/kustomization.yaml", expectErr: true},
		{
			name:    "shared cluster config patch",
			ignored: testWC + "/apps/patch_cluster_config.yaml",
			expectFiles: map[string]string{
				testWC + "/cluster/kustomization.yaml":            v2ClusterKustomization,
				testWC + "/cluster/patch_cluster_userconfig.yaml": v2ClusterUserConfigPatch,
				testWC + "/apps/kustomization.yaml":               v2AppsKustomization,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &afero.Afero{Fs: afero.NewMemMapFs()}
			writeFiles(t, fs, v1WorkloadCluster())
			env := Env{Ignored: func(path string) bool { return path == tc.ignored }}
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: testWC, StructureVersion: 1}

			changes, err := helmReleases.Apply(fs, testRepo, layer, env)
			checkIgnoredOutcome(t, tc.ignored, tc.expectErr, changes, err)

			want := map[string]string{tc.ignored: v1WorkloadCluster()[tc.ignored]}
			for path, expected := range tc.expectFiles {
				want[path] = expected
			}
			checkFiles(t, fs, want)
		})
	}
}

// checkIgnoredOutcome checks that the migration refused, saying git ignores
// the file, or else reported the ignored file as left.
func checkIgnoredOutcome(t *testing.T, ignored string, expectErr bool, changes []string, err error) {
	t.Helper()

	if expectErr {
		if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), "git ignores") {
			t.Fatalf("expected the ignored file refused, saying git ignores it, got %v, %v", changes, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(strings.Join(changes, "\n"), ignored+": left") {
		t.Errorf("expected the ignored file reported as left, got: %q", changes)
	}
}

// checkFiles checks the files hold what they should, byte for byte.
func checkFiles(t *testing.T, fs *afero.Afero, want map[string]string) {
	t.Helper()

	for path, expected := range want {
		got, _ := fs.ReadFile(filepath.Join(testRepo, path))
		if diff := cmp.Diff(expected, string(got)); diff != "" {
			t.Errorf("unexpected %s (-want +got):\n%s", path, diff)
		}
	}
}
