package wcluster

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// Test_runner_run_mapiCluster checks that adding a base to a cluster that
// still keeps its manifests in `mapi` is refused before anything is written:
// its Kustomization applies `mapi`, so Flux would never apply the cluster.
func Test_runner_run_mapiCluster(t *testing.T) {
	seed := map[string]string{
		layoutWCsDir + "/kustomization.yaml":          "resources:\n  - demowc.yaml\n",
		layoutWCsDir + "/demowc.yaml":                 strings.Replace(layoutSingleKustomization, "/demowc\"", "/demowc/mapi\"", 1),
		layoutWCDir + "/mapi/apps/kustomization.yaml": "resources: []\n",
	}
	fs := seedLayout(t, seed)

	err := runLayout(fs)
	if err == nil || !strings.Contains(err.Error(), "kubectl gs gitops upgrade") {
		t.Fatalf("expected the cluster to be refused, got: %v", err)
	}

	for _, path := range []string{layoutWCDir + "/kustomization.yaml", layoutWCDir + "/cluster", layoutWCDir + "/apps"} {
		if exists(t, fs, path) {
			t.Errorf("expected nothing written, found %s", path)
		}
	}
	for path, data := range seed {
		got, err := afero.ReadFile(fs, path)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if string(got) != data {
			t.Errorf("expected %s unchanged, got:\n%s", path, got)
		}
	}
}
