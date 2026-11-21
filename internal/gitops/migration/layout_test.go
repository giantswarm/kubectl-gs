package migration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	emptyResources = "resources: []\n"
	configMap      = "kind: ConfigMap\n"
	organization   = "kind: Organization\n"

	layoutWCs = testOrg + "/workload-clusters"
	layoutWC  = layoutWCs + "/demowc"

	// The files of a cluster as kubectl-gs generated it at structure
	// version 2, with the `mapi` directory.
	mapiWCKustomization = `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: demomc-clusters-demowc
  namespace: default
spec:
  interval: 1m
  path: "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi"
  prune: false
`
	mapiClusterKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/managed-by: flux
kind: Kustomization
patches:
  - path: patch_cluster_userconfig.yaml
resources:
  - ../../../../../../../../bases/clusters/capa
  - cluster_userconfig.yaml
`
	mapiAppKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../../../../../../bases/apps/hello-world
  - ../shared/values.yaml
`
)

func layoutRepo(t *testing.T, files map[string]string) *afero.Afero {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	writeFiles(t, fs, files)

	return fs
}

func v2MapiCluster() map[string]string {
	return map[string]string{
		layoutWCs + "/kustomization.yaml":                        "resources:\n  - demowc.yaml\n",
		layoutWCs + "/demowc.yaml":                               mapiWCKustomization,
		layoutWC + "/mapi/cluster/kustomization.yaml":            mapiClusterKustomization,
		layoutWC + "/mapi/cluster/cluster_userconfig.yaml":       configMap,
		layoutWC + "/mapi/cluster/patch_cluster_userconfig.yaml": "- op: add\n",
		layoutWC + "/mapi/apps/kustomization.yaml":               "resources:\n  - hello-world\n",
		layoutWC + "/mapi/apps/hello-world/kustomization.yaml":   mapiAppKustomization,
		layoutWC + "/mapi/apps/shared/values.yaml":               configMap,
		layoutWC + "/mapi/apps/hello-world/secret.dec.yaml":      "kind: Secret\n",
		layoutWC + "/out-of-band/configmaps/configmap.yaml":      configMap,
	}
}

// ignoring reports the decrypted Secret as ignored by git.
var ignoring = Env{Ignored: func(path string) bool { return strings.HasSuffix(path, ".dec.yaml") }}

func Test_Layout_MovesMapiUp(t *testing.T) {
	fs := layoutRepo(t, v2MapiCluster())
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	changes, err := version3.Apply(withEnv(fs, ignoring), testRepo, layer)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(changes) == 0 {
		t.Fatalf("expected the changes to be reported")
	}

	checkResources(t, fs, layoutWC+"/cluster/kustomization.yaml", []string{"../../../../../../../bases/clusters/capa", "cluster_userconfig.yaml"})
	checkResources(t, fs, layoutWC+"/apps/hello-world/kustomization.yaml", []string{"../../../../../../../../bases/apps/hello-world", "../shared/values.yaml"})
	checkResources(t, fs, layoutWC+"/kustomization.yaml", []string{"apps/", "cluster/"})
	checkPath(t, fs, layoutWCs+"/demowc.yaml", "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc")

	for _, f := range []string{"cluster/cluster_userconfig.yaml", "cluster/patch_cluster_userconfig.yaml", "apps/shared/values.yaml", "out-of-band/configmaps/configmap.yaml", "mapi/apps/hello-world/secret.dec.yaml"} {
		if ok, _ := fs.Exists(filepath.Join(testRepo, layoutWC, f)); !ok {
			t.Errorf("expected %s", f)
		}
	}
	if ok, _ := fs.Exists(filepath.Join(testRepo, layoutWC, "mapi/cluster")); ok {
		t.Errorf("expected mapi/cluster to be gone")
	}
	if !strings.Contains(strings.Join(changes, "\n"), "secret.dec.yaml") {
		t.Errorf("expected the ignored file left behind to be reported, got:\n%s", strings.Join(changes, "\n"))
	}

	again, err := version3.Apply(withEnv(fs, ignoring), testRepo, layer)
	if err != nil || len(again) != 0 {
		t.Errorf("expected a second run to change nothing, got %v, %v", again, err)
	}
}

// Test_Layout_AddsKustomization checks a cluster added with `--skip-mapi`,
// whose Kustomization applied its whole directory, `out-of-band` included.
func Test_Layout_AddsKustomization(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		layoutWC + "/cluster/kustomization.yaml":            emptyResources,
		layoutWC + "/apps/kustomization.yaml":               emptyResources,
		layoutWC + "/out-of-band/configmaps/configmap.yaml": configMap,
		layoutWC + "/extra.yaml":                            configMap,
	})
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkResources(t, fs, layoutWC+"/kustomization.yaml", []string{"apps/", "cluster/", "extra.yaml"})
	if !strings.Contains(strings.Join(changes, "\n"), "out-of-band") {
		t.Errorf("expected the change to out-of-band to be reported, got:\n%s", strings.Join(changes, "\n"))
	}
}

func Test_Layout_Refuses(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "out-of-band listed",
			files: map[string]string{
				layoutWC + "/kustomization.yaml":         "resources:\n  - cluster/\n  - out-of-band/\n",
				layoutWC + "/cluster/kustomization.yaml": emptyResources,
			},
		},
		{
			name: "both mapi/apps and apps",
			files: map[string]string{
				layoutWC + "/mapi/apps/kustomization.yaml": emptyResources,
				layoutWC + "/apps/kustomization.yaml":      emptyResources,
			},
		},
		{
			name: "directory without a kustomization.yaml",
			files: map[string]string{
				layoutWC + "/apps/kustomization.yaml": emptyResources,
				layoutWC + "/secrets/secret.enc.yaml": "kind: Secret\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := layoutRepo(t, tc.files)
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

			_, err := version3.Apply(fs, testRepo, layer)
			if !IsUnexpectedFile(err) {
				t.Fatalf("expected an unexpected file error, got: %v", err)
			}
		})
	}
}

func Test_Layout_Relocate(t *testing.T) {
	testCases := map[string]string{
		layoutWC + "/mapi/apps/hello-world": layoutWC + "/apps/hello-world",
		layoutWC + "/apps/hello-world":      layoutWC + "/apps/hello-world",
		layoutWC:                            layoutWC,
		testOrg + "/mapi/apps/x":            testOrg + "/mapi/apps/x",
	}
	for path, expected := range testCases {
		if got := version3.Relocate(path); got != expected {
			t.Errorf("expected %s to move to %s, got %s", path, expected, got)
		}
	}
}

func checkResources(t *testing.T, fs *afero.Afero, path string, expected []string) {
	t.Helper()

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	readInto(t, fs, path, &kustomization)
	if diff := cmp.Diff(expected, kustomization.Resources); diff != "" {
		t.Errorf("resources of %s (-want +got):\n%s", path, diff)
	}
}

func checkPath(t *testing.T, fs *afero.Afero, path, expected string) {
	t.Helper()

	var kustomization struct {
		Spec struct {
			Path string `json:"path"`
		} `json:"spec"`
	}
	readInto(t, fs, path, &kustomization)
	if kustomization.Spec.Path != expected {
		t.Errorf("expected %s to apply %s, got %s", path, expected, kustomization.Spec.Path)
	}
}

func readInto(t *testing.T, fs *afero.Afero, path string, into any) {
	t.Helper()

	data, err := fs.ReadFile(filepath.Join(testRepo, path))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	err = yaml.Unmarshal(data, into)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

// Test_Version3_Upgrade runs the registered migrations on a cluster with
// `mapi` and an app in it, and checks the app is recorded where it moved.
func Test_Version3_Upgrade(t *testing.T) {
	files := v2MapiCluster()
	files[testMC+"/demomc.yaml"] = namespaceMCKustomization
	files[testOrg+"/demoorg.yaml"] = organization
	fs := layoutRepo(t, files)

	md := metadata.New()
	md.StructureVersion = 2
	md.Layers = []metadata.Layer{
		{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2},
		{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2},
		{Kind: metadata.LayerApp, Path: layoutWC + "/mapi/apps/hello-world", StructureVersion: 2},
	}

	plan, err := NewPlan(md, Migrations, 3)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	plan.Env = ignoring
	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var got []string
	for _, l := range md.Layers {
		got = append(got, fmt.Sprintf("%s %s %d", l.Kind, l.Path, l.StructureVersion))
	}
	expected := []string{
		"organization " + testOrg + " 3",
		"workload-cluster " + layoutWC + " 3",
		"app " + layoutWC + "/apps/hello-world 3",
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("recorded layers (-want +got):\n%s", diff)
	}
}
