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

	changes, err := fixFluxPaths(fs, testRepo, layoutWC, Env{})
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

// markerCases put a Flux Kustomization among other documents with every way
// YAML marks where a document starts or ends.
func markerCases(kustomization string) map[string]struct {
	file string
	kept []string
} {
	const (
		first  = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: first\n"
		second = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: second\n"
	)
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

	return map[string]struct {
		file string
		kept []string
	}{
		"CRLF line endings": {
			file: crlf(first + "---\n" + kustomization + "---\n" + second),
			kept: []string{crlf(first + "---\n"), crlf("---\n" + second)},
		},
		"separator with a comment": {
			file: first + "--- # the Kustomization\n" + kustomization + "--- # the second\n" + second,
			kept: []string{first + "--- # the Kustomization\n", "--- # the second\n" + second},
		},
		"document end": {
			file: first + "...\n" + kustomization + "...\n---\n" + second,
			kept: []string{first + "...\n", "...\n---\n" + second},
		},
		"leading separator": {
			file: "--- # leading\n" + kustomization + "---\n" + first,
			kept: []string{"--- # leading\n", "---\n" + first},
		},
	}
}

// Test_FixFluxPaths_DocumentMarkers checks that the documents around a
// rewritten Flux Kustomization are kept byte for byte, whatever marks them.
func Test_FixFluxPaths_DocumentMarkers(t *testing.T) {
	for name, tc := range markerCases(mapiWCKustomization) {
		t.Run(name, func(t *testing.T) {
			fs := layoutRepo(t, map[string]string{layoutWCs + "/demowc.yaml": tc.file})

			changes, err := fixFluxPaths(fs, testRepo, layoutWC, Env{})
			if err != nil || len(changes) != 1 {
				t.Fatalf("expected one change, got %v, %v", changes, err)
			}

			data, err := fs.ReadFile(filepath.Join(testRepo, layoutWCs, "demowc.yaml"))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			checkMarkers(t, tc.file, string(data), tc.kept)
		})
	}
}

// checkMarkers checks that the documents around the rewritten Kustomization
// are kept, that it applies the cluster's directory, and that the line
// endings are kept.
func checkMarkers(t *testing.T, in, out string, kept []string) {
	t.Helper()

	for _, k := range kept {
		if !strings.Contains(out, k) {
			t.Errorf("expected %q kept, got:\n%q", k, out)
		}
	}
	if strings.Contains(out, "/mapi") || !strings.Contains(out, "workload-clusters/demowc") {
		t.Errorf("expected the Kustomization to apply the cluster's directory, got:\n%s", out)
	}
	if strings.Contains(in, "\r\n") && strings.Count(out, "\n") != strings.Count(out, "\r\n") {
		t.Errorf("expected CRLF line endings throughout, got:\n%q", out)
	}
}

// Test_FixFluxPaths_RefusesWhatItCantKeep checks that a Kustomization that
// starts on its `---` line is refused rather than rewritten, as re-encoding
// it would lose or repeat what that line holds.
func Test_FixFluxPaths_RefusesWhatItCantKeep(t *testing.T) {
	const flow = "--- {apiVersion: kustomize.toolkit.fluxcd.io/v1, kind: Kustomization, metadata: {name: demowc}, spec: {path: ./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi}}\n"
	fs := layoutRepo(t, map[string]string{layoutWCs + "/demowc.yaml": flow})

	_, err := fixFluxPaths(fs, testRepo, layoutWC, Env{})
	if !IsUnexpectedFile(err) {
		t.Fatalf("expected the file refused, got: %v", err)
	}

	data, _ := fs.ReadFile(filepath.Join(testRepo, layoutWCs, "demowc.yaml"))
	if string(data) != flow {
		t.Errorf("expected the file unchanged, got:\n%s", data)
	}
}

// Test_FixFluxPaths_LeavesIgnoredFiles checks, through a plan, that a Flux
// Kustomization applying `mapi` in a file git ignores, such as a backup next
// to the cluster's, is kept byte for byte, as git cannot restore it, while
// the cluster's own Kustomization next to it is still pointed at the
// cluster's directory.
func Test_FixFluxPaths_LeavesIgnoredFiles(t *testing.T) {
	files := v2MapiCluster()
	files[layoutWCs+"/backup.yaml"] = mapiWCKustomization
	fs := layoutRepo(t, files)

	md := metadata.New()
	md.StructureVersion = 2
	md.Layers = []metadata.Layer{{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}}

	plan, err := NewPlan(md, Migrations, 3)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	plan.Env = Env{Ignored: func(path string) bool {
		return path == layoutWCs+"/backup.yaml" || ignoring.IsIgnored(path)
	}}
	_, err = plan.Run(fs, testRepo, md, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	data, _ := fs.ReadFile(filepath.Join(testRepo, layoutWCs, "backup.yaml"))
	if diff := cmp.Diff(mapiWCKustomization, string(data)); diff != "" {
		t.Errorf("expected the ignored backup.yaml kept byte for byte (-want +got):\n%s", diff)
	}
	checkPath(t, fs, layoutWCs+"/demowc.yaml", "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc")
}
