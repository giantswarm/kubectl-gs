package check

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Test_Check_Version2IsBehind checks that a repository generated with
// structure version 2, which still has the `mapi` directory, is reported as
// behind, with the way to upgrade it.
func Test_Check_Version2IsBehind(t *testing.T) {
	fs := generatedRepo(t)
	out := new(bytes.Buffer)

	md := metadata.New()
	md.StructureVersion = 2
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2},
		{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2},
		{Kind: metadata.LayerWorkloadCluster, Path: testWC, StructureVersion: 2},
	}
	err := metadata.Save(&afero.Afero{Fs: fs}, testRepo, md)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	err = newTestRunner(fs, false, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err == nil {
		t.Fatal("expected a version 2 repository to be behind")
	}
	if !strings.Contains(err.Error(), "kubectl gs gitops upgrade --dry-run") {
		t.Errorf("expected the error to point at the upgrade, got: %s", err)
	}
	if !strings.Contains(out.String(), "3 layer(s) out of sync") {
		t.Errorf("expected every layer to be behind, got:\n%s", out.String())
	}
}
