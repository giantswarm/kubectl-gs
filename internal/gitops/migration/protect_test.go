package migration

import (
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	// wantPruneAnnotation is the prune annotation the tests expect, spelled
	// out for the same reason as wantPruneDisabled.
	wantPruneAnnotation = "kustomize.toolkit.fluxcd.io/prune"

	// wantPruneDisabled is the value of the prune annotation the tests expect.
	// It is spelled out rather than taken from pruneDisabled, so a change to
	// that constant fails them.
	wantPruneDisabled = "disabled"

	// reportedLeft is what the report of a file the upgrade leaves says.
	reportedLeft = "left"
)

type fluxKustomization struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
		Name        string            `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Path string `json:"path"`
	} `json:"spec"`
}

// Test_ManagementCluster_DisablesPruning checks that the upgrade gives the
// management cluster's Kustomization the prune: disabled annotation a newly
// generated one has, keeping its comments, and that a second run changes
// nothing.
func Test_ManagementCluster_DisablesPruning(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		testMC + "/demomc.yaml": "# the management cluster's own comment\n" + namespaceMCKustomization,
	})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(changes) != 1 || !strings.HasPrefix(changes[0], testMC+"/demomc.yaml: ") {
		t.Fatalf("expected one change to demomc.yaml, got %v, %v", changes, err)
	}

	var got fluxKustomization
	readInto(t, fs, testMC+"/demomc.yaml", &got)
	if got.Metadata.Annotations[wantPruneAnnotation] != wantPruneDisabled || got.Metadata.Name != "demomc-gitops" || got.Spec.Path != "./management-clusters/demomc" {
		t.Errorf("expected the annotation added and the rest kept, got %+v", got)
	}
	data, _ := fs.ReadFile(testRepo + "/" + testMC + "/demomc.yaml")
	if !strings.Contains(string(data), "# the management cluster's own comment") {
		t.Errorf("expected the comment kept, got:\n%s", data)
	}

	again, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(again) != 0 {
		t.Errorf("expected a second run to change nothing, got %v, %v", again, err)
	}
}

// Test_ManagementCluster_KeepsPruneAnnotation checks that a prune annotation
// the user set is never overwritten, and that the upgrade says so.
func Test_ManagementCluster_KeepsPruneAnnotation(t *testing.T) {
	annotated := strings.Replace(namespaceMCKustomization, "  name: demomc-gitops\n", "  annotations:\n    "+wantPruneAnnotation+": enabled\n  name: demomc-gitops\n", 1)
	fs := layoutRepo(t, map[string]string{testMC + "/demomc.yaml": annotated})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0], reportedLeft) {
		t.Fatalf("expected the file reported as left, got %v, %v", changes, err)
	}

	data, _ := fs.ReadFile(testRepo + "/" + testMC + "/demomc.yaml")
	if string(data) != annotated {
		t.Errorf("expected the file unchanged, got:\n%s", data)
	}
}

// Test_ManagementCluster_ProtectsOnlyItsRoot checks that only the
// Kustomization applying the management cluster's directory itself gets the
// prune: disabled annotation. Another in the same file, applying a directory
// under it, stays prunable, as a generated one is, and is kept byte for byte.
func Test_ManagementCluster_ProtectsOnlyItsRoot(t *testing.T) {
	apps := "---\n# the apps' own comment\n" + strings.NewReplacer(
		"  name: demomc-gitops\n", "  name: demomc-apps\n",
		"  path: ./management-clusters/demomc\n", "  path: ./management-clusters/demomc/apps\n",
	).Replace(namespaceMCKustomization)

	testCases := []struct {
		name string
		root string
	}{
		{name: "root as generated", root: "./management-clusters/demomc"},
		{name: "root with a trailing slash", root: "management-clusters/demomc/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := strings.Replace(namespaceMCKustomization, "  path: ./management-clusters/demomc\n", "  path: "+tc.root+"\n", 1)
			fs := layoutRepo(t, map[string]string{testMC + "/demomc.yaml": root + apps})
			layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

			changes, err := version3.Apply(fs, testRepo, layer, Env{})
			if err != nil || len(changes) != 1 {
				t.Fatalf("expected one change, got %v, %v", changes, err)
			}

			data, _ := fs.ReadFile(testRepo + "/" + testMC + "/demomc.yaml")
			rootOut, appsOut, found := strings.Cut(string(data), "---\n")
			if !found || "---\n"+appsOut != apps {
				t.Errorf("expected the apps Kustomization kept byte for byte, got:\n%s", data)
			}
			if !strings.Contains(rootOut, wantPruneAnnotation+": "+wantPruneDisabled) {
				t.Errorf("expected the root Kustomization annotated, got:\n%s", rootOut)
			}
		})
	}
}

// Test_ManagementCluster_NullAnnotations checks that annotations set to null,
// spelled out or left empty, are replaced with the prune annotation, keeping
// the comments, and that annotations that aren't a mapping are left and
// reported as such, never as protected.
func Test_ManagementCluster_NullAnnotations(t *testing.T) {
	testCases := []struct {
		name        string
		annotations string
		protected   bool
	}{
		{name: "null", annotations: "  annotations: null\n", protected: true},
		{name: "empty", annotations: "  annotations:\n", protected: true},
		{name: "tilde", annotations: "  annotations: ~\n", protected: true},
		{name: "sequence", annotations: "  annotations: []\n", protected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkAnnotations(t, tc.annotations, tc.protected)
		})
	}
}

// checkAnnotations upgrades a management cluster whose Kustomization has the
// given annotations, and checks it is protected, or left and reported so.
func checkAnnotations(t *testing.T, annotations string, protected bool) {
	t.Helper()

	original := strings.Replace(namespaceMCKustomization, "metadata:\n", "metadata:\n  # the annotations' own comment\n"+annotations, 1)
	fs := layoutRepo(t, map[string]string{testMC + "/demomc.yaml": original})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected one change, got %v, %v", changes, err)
	}
	data, _ := fs.ReadFile(testRepo + "/" + testMC + "/demomc.yaml")

	if !protected {
		if !strings.Contains(changes[0], reportedLeft) || string(data) != original {
			t.Errorf("expected the file left and reported so, got %v:\n%s", changes, data)
		}
		return
	}

	var got fluxKustomization
	readInto(t, fs, testMC+"/demomc.yaml", &got)
	if got.Metadata.Annotations[wantPruneAnnotation] != wantPruneDisabled || !strings.Contains(changes[0], "annotated") {
		t.Errorf("expected the annotation added and reported, got %v:\n%s", changes, data)
	}
	if !strings.Contains(string(data), "# the annotations' own comment") {
		t.Errorf("expected the comment kept, got:\n%s", data)
	}

	again, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(again) != 0 {
		t.Errorf("expected a second run to change nothing, got %v, %v", again, err)
	}
}

// Test_Version3_LeavesIgnoredFiles checks, through a plan, that the upgrade
// never changes the files git ignores, which git cannot restore: a copy of a
// cluster's Kustomization, and the management cluster's Kustomization.
func Test_Version3_LeavesIgnoredFiles(t *testing.T) {
	files := v2MapiCluster()
	files[layoutWCs+"/backup.yaml"] = mapiWCKustomization
	files[testMC+"/demomc.yaml"] = namespaceMCKustomization
	fs := layoutRepo(t, files)

	md := metadata.New()
	md.StructureVersion = 2
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2},
		{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2},
	}

	plan, err := NewPlan(md, Migrations, 3)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	plan.Env = Env{Ignored: func(path string) bool {
		return path == layoutWCs+"/backup.yaml" || path == testMC+"/demomc.yaml" || ignoring.IsIgnored(path)
	}}
	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for _, f := range []string{layoutWCs + "/backup.yaml", testMC + "/demomc.yaml"} {
		data, _ := fs.ReadFile(testRepo + "/" + f)
		if string(data) != files[f] {
			t.Errorf("expected the ignored %s kept byte for byte, got:\n%s", f, data)
		}
	}
}
