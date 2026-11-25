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

	oldGate = "status.conditions.exists(c, c.type == 'Available' && c.status == 'True')"
	newGate = "status.conditions.exists(c, c.type == 'ControlPlaneAvailable' && c.status == 'True') && status.conditions.exists(c, c.type == 'RemoteConnectionProbe' && c.status == 'True')"

	// gatedWCKustomization is a cluster's Kustomization as kubectl-gs
	// generated it with a cluster base, before the move out of `mapi`.
	gatedWCKustomization = `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: demomc-clusters-demowc
  namespace: default
spec:
  # the cluster's own comment
  healthCheckExprs:
    - apiVersion: cluster.x-k8s.io/v1beta2
      kind: Cluster
      current: GATE
  healthChecks:
    - apiVersion: cluster.x-k8s.io/v1beta2
      kind: Cluster
      name: demowc
  interval: 1m
  path: "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi"
  prune: false
`
)

type fluxKustomization struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
		Name        string            `json:"name"`
	} `json:"metadata"`
	Spec struct {
		HealthCheckExprs []struct {
			Current string `json:"current"`
		} `json:"healthCheckExprs"`
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

// Test_WorkloadCluster_GatesOnControlPlane checks that the upgrade replaces
// the health check kubectl-gs used to write, on the Cluster being Available,
// with the control plane gate, and leaves one the user changed.
func Test_WorkloadCluster_GatesOnControlPlane(t *testing.T) {
	testCases := []struct {
		name     string
		gate     string
		expected string
		change   string
	}{
		{name: "generated gate", gate: oldGate, expected: newGate, change: "control plane"},
		{name: "customised gate", gate: "status.conditions.exists(c, c.type == 'Ready')", expected: "status.conditions.exists(c, c.type == 'Ready')", change: reportedLeft},
		// An entry with expressions kubectl-gs never wrote is the user's as
		// a whole, though its current is the one kubectl-gs wrote.
		{name: "generated gate with a failed expression", gate: oldGate + "\n      failed: status.conditions.exists(c, c.type == 'Failed')", expected: oldGate, change: reportedLeft},
		{name: "generated gate with an inProgress expression", gate: oldGate + "\n      inProgress: status.phase == 'Provisioning'", expected: oldGate, change: reportedLeft},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkGate(t, tc.gate, tc.expected, tc.change)
		})
	}
}

// checkGate upgrades a cluster whose Kustomization has the given gate, and
// checks the gate, the report, the comment and a second run.
func checkGate(t *testing.T, gate, expected, change string) {
	t.Helper()

	files := v2MapiCluster()
	files[layoutWCs+"/demowc.yaml"] = strings.Replace(gatedWCKustomization, "GATE", gate, 1)
	fs := layoutRepo(t, files)
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}
	file := layoutWCs + "/demowc.yaml"

	original := strings.Replace(files[file], "/demowc/mapi\"", "/demowc\"", 1)

	changes, err := version3.Apply(fs, testRepo, layer, ignoring)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !containsChange(changes, file+": ", change) {
		t.Errorf("expected demowc.yaml reported with %q, got %v", change, changes)
	}
	if data, _ := fs.ReadFile(testRepo + "/" + file); change == reportedLeft && string(data) != original {
		t.Errorf("expected a health check left to keep the file as it was but for its path, got:\n%s", data)
	}

	var got fluxKustomization
	readInto(t, fs, file, &got)
	if len(got.Spec.HealthCheckExprs) != 1 || got.Spec.HealthCheckExprs[0].Current != expected {
		t.Errorf("expected the gate %q, got %+v", expected, got.Spec.HealthCheckExprs)
	}
	before, _ := fs.ReadFile(testRepo + "/" + file)
	if !strings.Contains(string(before), "# the cluster's own comment") {
		t.Errorf("expected the comment kept, got:\n%s", before)
	}

	again, err := version3.Apply(fs, testRepo, layer, ignoring)
	after, _ := fs.ReadFile(testRepo + "/" + file)
	if err != nil || containsChange(again, file+": ", "control plane") || string(after) != string(before) {
		t.Errorf("expected a second run to change nothing, got %v, %v:\n%s", again, err, after)
	}
}

func containsChange(changes []string, prefix, text string) bool {
	for _, c := range changes {
		if strings.HasPrefix(c, prefix) && strings.Contains(c, text) {
			return true
		}
	}

	return false
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
// cluster's Kustomization with the old health check, and the management
// cluster's Kustomization.
func Test_Version3_LeavesIgnoredFiles(t *testing.T) {
	backup := strings.Replace(strings.Replace(gatedWCKustomization, "GATE", oldGate, 1), "/demowc/mapi\"", "/demowc\"", 1)
	files := v2MapiCluster()
	files[layoutWCs+"/backup.yaml"] = backup
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
