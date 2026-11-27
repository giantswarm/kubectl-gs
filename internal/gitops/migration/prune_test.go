package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const pruneDisabledAnnotations = "commonAnnotations:\n  kustomize.toolkit.fluxcd.io/prune: disabled\n"

// prunedMapiCluster is a cluster with `mapi`, whose Kustomization prunes.
func prunedMapiCluster() map[string]string {
	files := v2MapiCluster()
	files[layoutWCs+"/demowc.yaml"] = strings.Replace(mapiWCKustomization, "prune: false", "prune: true", 1)

	return files
}

// upgradePrunedMapi runs the upgrade on a cluster whose Kustomization
// prunes mapi, with the given files added, and returns its error.
func upgradePrunedMapi(t *testing.T, extra map[string]string) (func() bool, error) {
	t.Helper()

	files := prunedMapiCluster()
	for k, v := range extra {
		files[k] = v
	}
	fs := layoutRepo(t, files)
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, ignoring)
	mapiKept := func() bool {
		ok, _ := fs.DirExists(filepath.Join(testRepo, layoutWC, "mapi", "cluster"))
		return ok
	}

	return mapiKept, err
}

// Test_Layout_RefusesPrunedMapi checks that a cluster whose Kustomization
// applies mapi with prune: true isn't upgraded while an object it applies
// from there lacks kustomize.toolkit.fluxcd.io/prune: disabled, as the
// upgrade docs say to add first.
func Test_Layout_RefusesPrunedMapi(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		refused string
	}{
		{
			name:    "nothing protected",
			refused: "mapi/apps/kustomization.yaml",
		},
		{
			name: "only the cluster protected",
			files: map[string]string{
				layoutWC + "/mapi/cluster/kustomization.yaml": mapiClusterKustomization + pruneDisabledAnnotations,
			},
			refused: "mapi/apps/kustomization.yaml",
		},
		{
			name: "an object without the annotation next to protected kustomizations",
			files: map[string]string{
				layoutWC + "/mapi/cluster/kustomization.yaml": mapiClusterKustomization + pruneDisabledAnnotations,
				layoutWC + "/mapi/apps/kustomization.yaml":    "resources:\n  - hello-world\n" + pruneDisabledAnnotations,
				layoutWC + "/mapi/loose.yaml":                 configMap,
			},
			refused: "mapi/loose.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapiKept, err := upgradePrunedMapi(t, tc.files)
			if !IsUnexpectedFile(err) {
				t.Fatalf("expected the upgrade refused, got: %v", err)
			}
			for _, want := range []string{layoutWCs + "/demowc.yaml", tc.refused, "kustomize.toolkit.fluxcd.io/prune: disabled"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected the error to name %q, got: %s", want, err)
				}
			}
			if !mapiKept() {
				t.Errorf("expected mapi left as it was")
			}
		})
	}
}

// Test_Layout_UpgradesProtectedPrunedMapi checks that a cluster whose
// Kustomization prunes mapi is upgraded once every object it applies from
// there is annotated, whichever way.
func Test_Layout_UpgradesProtectedPrunedMapi(t *testing.T) {
	testCases := map[string]map[string]string{
		"every kustomization protected": {
			layoutWC + "/mapi/cluster/kustomization.yaml": mapiClusterKustomization + pruneDisabledAnnotations,
			layoutWC + "/mapi/apps/kustomization.yaml":    "resources:\n  - hello-world\n" + pruneDisabledAnnotations,
		},
		"mapi's own kustomization protected": {
			layoutWC + "/mapi/kustomization.yaml": "resources:\n  - apps/\n  - cluster/\n" + pruneDisabledAnnotations,
		},
		"children protected one by one": {
			layoutWC + "/mapi/cluster/kustomization.yaml":          mapiClusterKustomization + pruneDisabledAnnotations,
			layoutWC + "/mapi/apps/kustomization.yaml":             "resources:\n  - hello-world\n  - annotated.yaml\n",
			layoutWC + "/mapi/apps/annotated.yaml":                 "kind: ConfigMap\nmetadata:\n  annotations:\n    kustomize.toolkit.fluxcd.io/prune: disabled\n",
			layoutWC + "/mapi/apps/hello-world/kustomization.yaml": "resources:\n  - ../shared/values.yaml\n" + pruneDisabledAnnotations,
		},
	}

	for name, files := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := upgradePrunedMapi(t, files)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
