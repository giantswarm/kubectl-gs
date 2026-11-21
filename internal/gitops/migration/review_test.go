package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// Test_FixKustomizationPaths checks every kustomization field naming a path
// that leads out of the moved directory.
func Test_FixKustomizationPaths(t *testing.T) {
	const moved = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
bases:
  - ../../base
components:
  - ../../components/region
configMapGenerator:
  - files:
      - values=../../values.yaml
      - local.yaml
    envs:
      - ../../app.env
    name: config
crds:
  - ../../crds.yaml
generators:
  - ../../generator.yaml
patches:
  - path: ../../patch.yaml
patchesJson6902:
  - path: ../../json-patch.yaml
patchesStrategicMerge:
  - ../../merge.yaml
replacements:
  - path: ../../replacement.yaml
resources:
  - ../../resource.yaml
  - ../sibling
  - https://example.com/remote.yaml
secretGenerator:
  - env: ../../secret.env
    name: secret
transformers:
  - ../../transformer.yaml
`
	fs := layoutRepo(t, map[string]string{layoutWC + "/cluster/kustomization.yaml": moved})
	oldDir := filepath.Join(testRepo, layoutWC, "mapi", "cluster")
	newFile := filepath.Join(testRepo, layoutWC, "cluster", "kustomization.yaml")

	changes, err := fixKustomizationPaths(fs, testRepo, filepath.Join(testRepo, layoutWC, "mapi"), oldDir, newFile)
	if err != nil || len(changes) != 1 {
		t.Fatalf("expected one change, got %v, %v", changes, err)
	}

	data, err := fs.ReadFile(newFile)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	got := string(data)
	for _, p := range []string{"base", "components/region", "values=../values.yaml", "app.env", "crds.yaml", "generator.yaml", "patch.yaml", "json-patch.yaml", "merge.yaml", "replacement.yaml", "resource.yaml", "secret.env", "transformer.yaml"} {
		want := "../" + strings.TrimPrefix(p, "values=../")
		if strings.HasPrefix(p, "values=") {
			want = p
		}
		if !strings.Contains(got, " "+want+"\n") && !strings.Contains(got, ": "+want+"\n") {
			t.Errorf("expected %s one level shorter, got:\n%s", p, got)
		}
	}
	for _, kept := range []string{"- local.yaml", "- ../sibling", "https://example.com/remote.yaml"} {
		if !strings.Contains(got, kept) {
			t.Errorf("expected %s kept, got:\n%s", kept, got)
		}
	}
}

// Test_FixFluxPaths_MultipleDocuments checks that only the Flux
// Kustomizations of a file with several documents are rewritten, and the
// other documents are kept as they are, in .yaml and .yml files.
func Test_FixFluxPaths_MultipleDocuments(t *testing.T) {
	const (
		configMap = "# keep this comment\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: notes   # and this one\ndata:\n  path: ./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi\n"
		other     = "apiVersion: kustomize.toolkit.fluxcd.io/v1\nkind: Kustomization\nmetadata:\n  name: other\nspec:\n  path: ./elsewhere\n"
	)
	multi := configMap + "---\n" + mapiWCKustomization + "---\n" + other
	fs := layoutRepo(t, map[string]string{
		layoutWCs + "/demowc.yaml":     multi,
		layoutWCs + "/demowc-apps.yml": mapiWCKustomization,
	})

	changes, err := fixFluxPaths(fs, testRepo, layoutWC)
	if err != nil || len(changes) != 2 {
		t.Fatalf("expected two changes, got %v, %v", changes, err)
	}

	data, err := fs.ReadFile(filepath.Join(testRepo, layoutWCs, "demowc.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	documents := strings.Split(string(data), "---\n")
	if len(documents) != 3 {
		t.Fatalf("expected three documents, got:\n%s", data)
	}
	if diff := cmp.Diff(configMap, documents[0]); diff != "" {
		t.Errorf("first document (-want +got):\n%s", diff)
	}
	if strings.Contains(documents[1], "/mapi") || !strings.Contains(documents[1], "workload-clusters/demowc") {
		t.Errorf("expected the Kustomization to apply the cluster's directory, got:\n%s", documents[1])
	}
	if diff := cmp.Diff(other, documents[2]); diff != "" {
		t.Errorf("last document (-want +got):\n%s", diff)
	}
	checkPath(t, fs, layoutWCs+"/demowc-apps.yml", "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc")
}

// Test_Layout_RefusesPrunedOutOfBand checks that a cluster whose
// Kustomization applied its whole directory with prune: true isn't upgraded:
// out-of-band leaves what it applies, and Flux would delete from the
// management cluster what it applied from there.
func Test_Layout_RefusesPrunedOutOfBand(t *testing.T) {
	pruned := strings.Replace(strings.Replace(mapiWCKustomization, "/mapi\"", "\"", 1), "prune: false", "prune: true", 1)
	fs := layoutRepo(t, map[string]string{
		layoutWCs + "/demowc.yaml":                          pruned,
		layoutWC + "/apps/kustomization.yaml":               emptyResources,
		layoutWC + "/out-of-band/configmaps/configmap.yaml": configMap,
	})
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), "prune: false") {
		t.Fatalf("expected the upgrade refused with the way out, got: %v", err)
	}
	if ok, _ := fs.Exists(filepath.Join(testRepo, layoutWC, "kustomization.yaml")); ok {
		t.Errorf("expected nothing written")
	}
}

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

func Test_Version3_ManualStepsPointAtTheTemplate(t *testing.T) {
	for _, s := range []string{"prune: true", "docs/how-to/upgrade-fork.md#move-the-workload-cluster-kustomizations", "three commits"} {
		if !strings.Contains(version3.ManualSteps, s) {
			t.Errorf("expected the manual steps to mention %q", s)
		}
	}
	if strings.Contains(version3.ManualSteps, "sourceRef") {
		t.Errorf("expected no paraphrase of the steps, got:\n%s", version3.ManualSteps)
	}
}

// Test_Namespace_NamesTheKustomization checks the error for an organisation
// directory the organisation's kustomization.yaml can't list.
func Test_Namespace_NamesTheKustomization(t *testing.T) {
	fs := layoutRepo(t, map[string]string{
		testMC + "/demomc.yaml":              namespaceMCKustomization,
		testOrg + "/demoorg.yaml":            organization,
		testOrg + "/secrets/secret.enc.yaml": secret,
	})
	layer := metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), "organisation's kustomization.yaml") {
		t.Fatalf("expected the organisation's kustomization.yaml named, got: %v", err)
	}
}
