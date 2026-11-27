package migration

import (
	"path/filepath"
	"strings"
	"testing"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	appsResource    = "apps/"
	clusterResource = "cluster/"
	orgFile         = "demoorg.yaml"
)

// Test_Layout_RefusesClusterKustomizationNextToMapi checks that a cluster
// with a kustomization file of its own next to `mapi` isn't upgraded: the
// file doesn't list what moves up from `mapi`, so the cluster's Kustomization
// would stop applying it, and with prune: true Flux would delete it.
func Test_Layout_RefusesClusterKustomizationNextToMapi(t *testing.T) {
	for _, name := range []string{"kustomization.yaml", "kustomization.yml", "Kustomization"} {
		t.Run(name, func(t *testing.T) {
			files := v2MapiCluster()
			files[layoutWC+"/"+name] = "resources:\n  - other.yaml\n"
			files[layoutWC+"/other.yaml"] = configMap
			fs := layoutRepo(t, files)
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

			_, err := version3.Apply(fs, testRepo, layer, ignoring)
			if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "apps") {
				t.Fatalf("expected the upgrade refused, naming %s and what moves, got: %v", name, err)
			}
			if ok, _ := fs.DirExists(filepath.Join(testRepo, layoutWC, "mapi", "apps")); !ok {
				t.Errorf("expected mapi left as it was")
			}
		})
	}
}

// Test_Layout_KeepsMapiKustomizationName checks that a `mapi` kustomization
// file moves up as the cluster's, whatever kustomize name it has, and that
// no second one is written next to it.
func Test_Layout_KeepsMapiKustomizationName(t *testing.T) {
	for _, name := range []string{"kustomization.yml", "Kustomization"} {
		t.Run(name, func(t *testing.T) {
			files := v2MapiCluster()
			files[layoutWC+"/mapi/"+name] = "resources:\n  - apps/\n  - cluster/\n"
			fs := layoutRepo(t, files)
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

			_, err := version3.Apply(fs, testRepo, layer, ignoring)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			checkResources(t, fs, layoutWC+"/"+name, []string{appsResource, clusterResource})
			if ok, _ := fs.Exists(filepath.Join(testRepo, layoutWC, "kustomization.yaml")); ok {
				t.Errorf("expected no kustomization.yaml next to %s", name)
			}
		})
	}
}

// Test_Layout_ListsDirectoryWithKustomizationYml checks that a directory
// whose kustomization file is a kustomization.yml is listed, not refused.
func Test_Layout_ListsDirectoryWithKustomizationYml(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		layoutWC + "/cluster/kustomization.yaml": emptyResources,
		layoutWC + "/apps/kustomization.yml":     emptyResources,
		layoutWC + "/apps/hello.yaml":            configMap,
	})
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkResources(t, fs, layoutWC+"/kustomization.yaml", []string{appsResource, clusterResource})
}

// Test_ManagementCluster_KeepsSecretsKustomizationYml checks that secrets
// with a kustomization.yml get no kustomization.yaml next to it.
func Test_ManagementCluster_KeepsSecretsKustomizationYml(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		testMC + "/secrets/kustomization.yml":      "resources:\n  - demomc.gpgkey.enc.yaml\n",
		testMC + "/secrets/demomc.gpgkey.enc.yaml": secret,
	})
	layer := metadata.Layer{Kind: metadata.LayerManagementCluster, Path: testMC, StructureVersion: 2}

	changes, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil || len(changes) != 0 {
		t.Fatalf("expected nothing changed, got %v, %v", changes, err)
	}
	if ok, _ := fs.Exists(filepath.Join(testRepo, testMC, "secrets", "kustomization.yaml")); ok {
		t.Errorf("expected no kustomization.yaml next to kustomization.yml")
	}
}

// Test_Namespace_KeepsKustomizationYml checks that an organisation with a
// kustomization.yml gets namespace.yaml listed there, and that a namespace
// directory with one is left as it is.
func Test_Namespace_KeepsKustomizationYml(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		testMC + "/demomc.yaml":                           namespaceMCKustomization,
		testOrg + "/" + orgFile:                           organization,
		testOrg + "/kustomization.yml":                    "resources:\n  - demoorg.yaml\n  - workload-clusters\n",
		testOrg + "/namespace/kustomization.yml":          emptyResources,
		testOrg + "/workload-clusters/kustomization.yaml": emptyResources,
	})
	layer := metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkResources(t, fs, testOrg+"/kustomization.yml", []string{orgFile, gitopskey.WorkloadClustersDirName(), gitopskey.NamespaceFileName()})
	for _, f := range []string{testOrg + "/kustomization.yaml", testOrg + "/namespace/kustomization.yaml"} {
		if ok, _ := fs.Exists(filepath.Join(testRepo, f)); ok {
			t.Errorf("expected no %s next to a kustomization.yml", f)
		}
	}
}
