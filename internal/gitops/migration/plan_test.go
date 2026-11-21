package migration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Test_Plan_RelocatesLayers checks that a layer a migration moves, such as
// an app in a workload cluster's `mapi` directory, is recorded, and passed to
// later migrations, at its new path, though no migration of its own ran.
func Test_Plan_RelocatesLayers(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	md := metadata.New()
	md.StructureVersion = 1
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerApp, Path: testOrg + "/old/app", StructureVersion: 1},
	}

	rec := &recording{}
	moving := rec.migration(1, metadata.LayerWorkloadCluster)
	moving.Relocate = func(path string) string {
		return strings.Replace(path, "/old/", "/new/", 1)
	}

	plan, err := NewPlan(md, []Migration{moving, rec.migration(2, metadata.LayerApp)}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := []string{"2:" + testOrg + "/new/app@2"}
	if fmt.Sprint(rec.steps) != fmt.Sprint(want) {
		t.Errorf("expected steps %v, got %v", want, rec.steps)
	}
	if len(md.Layers) != 1 || md.Layers[0].Path != testOrg+"/new/app" || md.Layers[0].StructureVersion != 3 {
		t.Errorf("expected the app recorded at its new path and version 3, got %+v", md.Layers)
	}
}

// Test_Plan_PassesEnv checks that the migrations see which files git
// ignores, as they must leave those alone.
func Test_Plan_PassesEnv(t *testing.T) {
	fs, md := testRepository(t)

	seen := false
	m := Migration{
		From:        1,
		Description: "checks the environment",
		Kinds:       []string{metadata.LayerOrganization},
		Apply: func(fs *afero.Afero, _ string, _ metadata.Layer) ([]string, error) {
			seen = EnvOf(fs).IsIgnored("secret.yaml")
			return nil, nil
		},
	}

	plan, err := NewPlan(md, []Migration{m}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	plan.Env = Env{Ignored: func(path string) bool { return path == "secret.yaml" }}

	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !seen {
		t.Errorf("expected the migration to see the ignored files")
	}
}
