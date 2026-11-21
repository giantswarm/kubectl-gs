package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Test_Layout_RefusesPrunedOutOfBand checks that a cluster whose
// Kustomization applied its whole directory with prune: true isn't upgraded:
// out-of-band leaves what it applies, and Flux would delete from the
// management cluster what it applied from there.
func Test_Layout_RefusesPrunedOutOfBand(t *testing.T) {
	pruned := strings.Replace(strings.Replace(mapiWCKustomization, "/mapi\"", "\"", 1), "prune: false", "prune: true", 1)
	fs := layoutRepo(t, map[string]string{
		layoutWCs + "/demowc.yaml":                          pruned,
		layoutWC + "/apps/kustomization.yaml":               emptyResources,
		layoutWC + "/out-of-band/configmaps/configmap.yaml": configMap,
	})
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), "prune: false") {
		t.Fatalf("expected the upgrade refused with the way out, got: %v", err)
	}
	if ok, _ := fs.Exists(filepath.Join(testRepo, layoutWC, "kustomization.yaml")); ok {
		t.Errorf("expected nothing written")
	}
}

// Test_Layout_RefusesPrunedOutOfBand_DocumentMarkers checks that a pruning
// Kustomization is found among other documents, whatever marks them.
func Test_Layout_RefusesPrunedOutOfBand_DocumentMarkers(t *testing.T) {
	pruned := strings.Replace(strings.Replace(mapiWCKustomization, "/mapi\"", "\"", 1), "prune: false", "prune: true", 1)
	for name, tc := range markerCases(pruned) {
		t.Run(name, func(t *testing.T) {
			fs := layoutRepo(t, map[string]string{
				layoutWCs + "/demowc.yaml":                          tc.file,
				layoutWC + "/apps/kustomization.yaml":               emptyResources,
				layoutWC + "/out-of-band/configmaps/configmap.yaml": configMap,
			})
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

			_, err := version3.Apply(fs, testRepo, layer, Env{})
			if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), "prune: false") {
				t.Fatalf("expected the upgrade refused, got: %v", err)
			}
		})
	}
}
