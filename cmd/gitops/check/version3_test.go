package check

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
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

// Test_Check_AdoptPrintsRecordedVersions checks that adopting a repository
// laid out as structure version 2 prints the version each layer is recorded
// at, rather than claiming the current one.
func Test_Check_AdoptPrintsRecordedVersions(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	files := map[string]string{
		testMC + "/secrets/demo.enc.yaml":           "kind: Secret\n",
		testWC + "/mapi/cluster/kustomization.yaml": "resources: []\n",
	}
	for path, data := range files {
		err := fs.WriteFile(testRepo+"/"+path, []byte(data), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	out := new(bytes.Buffer)

	err := newTestRunner(fs.Fs, true, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	md, err := metadata.Load(fs, testRepo)
	if err != nil {
		t.Fatalf("expected adoption to write the metadata file: %s", err)
	}
	if len(md.Layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(md.Layers))
	}

	printed := out.String()
	if strings.Contains(printed, fmt.Sprintf("structure version %d", metadata.StructureVersion)) {
		t.Errorf("expected no claim of version %d, got:\n%s", metadata.StructureVersion, printed)
	}
	if strings.Contains(printed, "assumes these layers are current") || !strings.Contains(printed, "from its layout") || !strings.Contains(printed, "not detectable") {
		t.Errorf("expected the note to describe the inferred version and warn of earlier drift, got:\n%s", printed)
	}
	for _, l := range md.Layers {
		if l.StructureVersion != 2 {
			t.Errorf("expected %s to be recorded at version 2, got %d", l.Kind, l.StructureVersion)
		}
		pattern := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(l.Kind) + `\s+2\s+1$`)
		if !pattern.MatchString(printed) {
			t.Errorf("expected %s to be printed at version 2, got:\n%s", l.Kind, printed)
		}
	}
}

// Test_Check_NotUpToDateWithLayersBehind checks that the repository's summary
// line does not call it up to date while a recorded layer is behind, as
// after adopting a repository laid out as structure version 2: the
// repository level version records the newest kubectl-gs that wrote to it,
// not the oldest layer.
func Test_Check_NotUpToDateWithLayersBehind(t *testing.T) {
	fs := generatedRepo(t)

	md := metadata.New()
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerWorkloadCluster, Path: testWC, StructureVersion: 2},
	}
	err := metadata.Save(&afero.Afero{Fs: fs}, testRepo, md)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	out := new(bytes.Buffer)
	err = newTestRunner(fs, false, out).run(context.Background(), newTestCommand(testRepo, false), nil)
	if err == nil {
		t.Fatal("expected a layer behind to fail the check")
	}
	if strings.Contains(out.String(), "(up to date)") {
		t.Errorf("expected no up to date summary while a layer is behind, got:\n%s", out.String())
	}
}
