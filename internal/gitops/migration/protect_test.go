package migration

import (
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	pruneAnnotation = "kustomize.toolkit.fluxcd.io/prune"

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
	if got.Metadata.Annotations[pruneAnnotation] != "disabled" || got.Metadata.Name != "demomc-gitops" || got.Spec.Path != "./management-clusters/demomc" {
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
	annotated := strings.Replace(namespaceMCKustomization, "  name: demomc-gitops\n", "  annotations:\n    "+pruneAnnotation+": enabled\n  name: demomc-gitops\n", 1)
	fs := layoutRepo(t, map[string]string{testMC + "/demomc.yaml": annotated})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0], "left") {
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
		{name: "customised gate", gate: "status.conditions.exists(c, c.type == 'Ready')", expected: "status.conditions.exists(c, c.type == 'Ready')", change: "left"},
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

	changes, err := version3.Apply(fs, testRepo, layer, ignoring)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !containsChange(changes, file+": ", change) {
		t.Errorf("expected demowc.yaml reported with %q, got %v", change, changes)
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
