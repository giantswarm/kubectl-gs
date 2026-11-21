package migration

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Test_ManagementCluster_ProtectsSecrets checks that the management
// cluster's secrets get a kustomization.yaml listing every key there, which
// keeps Flux from deleting them once the keys of the workload clusters move.
func Test_ManagementCluster_ProtectsSecrets(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		testMC + "/secrets/demomc.gpgkey.enc.yaml": secret,
		testMC + "/secrets/demowc.gpgkey.enc.yaml": secret,
		testMC + "/secrets/plain.dec.yaml":         secret,
	})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, ignoring)
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected one change, got %v, %v", changes, err)
	}

	var kustomization struct {
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		Labels            []struct {
			Pairs map[string]string `json:"pairs"`
		} `json:"labels"`
		Resources []string `json:"resources"`
	}
	readInto(t, fs, testMC+"/secrets/kustomization.yaml", &kustomization)
	if diff := cmp.Diff([]string{"demomc.gpgkey.enc.yaml", "demowc.gpgkey.enc.yaml"}, kustomization.Resources); diff != "" {
		t.Errorf("resources (-want +got):\n%s", diff)
	}
	if kustomization.CommonAnnotations["kustomize.toolkit.fluxcd.io/prune"] != "disabled" || len(kustomization.Labels) != 1 || kustomization.Labels[0].Pairs["giantswarm.io/prevent-deletion"] != "true" {
		t.Errorf("expected the keys protected, got %+v", kustomization)
	}

	again, err := version3.Apply(fs, testRepo, layer, ignoring)
	if err != nil || len(again) != 0 {
		t.Errorf("expected a second run to change nothing, got %v, %v", again, err)
	}
}
