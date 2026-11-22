package migration

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const namespaceMCKustomization = `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-master
  path: ./management-clusters/demomc
  sourceRef:
    kind: GitRepository
    name: gitops-demo
`

// Test_Namespace_AddsNamespaceKustomization checks that an organisation gets
// its namespace Kustomization, applying an empty namespace/ for now, while its
// workload clusters' Kustomizations stay in `default`: moving them takes
// steps Flux has to apply in order.
func Test_Namespace_AddsNamespaceKustomization(t *testing.T) {
	testCases := []struct {
		name     string
		existing string
		expected []string
	}{
		{
			name:     "organisation without a kustomization.yaml",
			expected: []string{"demoorg.yaml", "namespace.yaml", "workload-clusters"},
		},
		{
			name:     "organisation with a kustomization.yaml",
			existing: "resources:\n  - demoorg.yaml\n  - workload-clusters\n",
			expected: []string{"demoorg.yaml", "workload-clusters", "namespace.yaml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				testMC + "/demomc.yaml":                           namespaceMCKustomization,
				testOrg + "/demoorg.yaml":                         organization,
				testOrg + "/workload-clusters/kustomization.yaml": emptyResources,
			}
			if tc.existing != "" {
				files[testOrg+"/kustomization.yaml"] = tc.existing
			}
			fs := layoutRepo(t, files)
			layer := metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2}

			changes, err := version3.Apply(fs, testRepo, layer, Env{})
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(changes) == 0 {
				t.Fatalf("expected the changes to be reported")
			}

			checkResources(t, fs, testOrg+"/kustomization.yaml", tc.expected)
			checkResources(t, fs, testOrg+"/namespace/kustomization.yaml", []string{})
			checkNamespaceKustomization(t, fs)

			again, err := version3.Apply(fs, testRepo, layer, Env{})
			if err != nil || len(again) != 0 {
				t.Errorf("expected a second run to change nothing, got %v, %v", again, err)
			}
		})
	}
}

func Test_Namespace_RefusesWithoutManagementCluster(t *testing.T) {
	fs := layoutRepo(t, map[string]string{testOrg + "/demoorg.yaml": organization})
	layer := metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, Env{})
	if !IsUnexpectedFile(err) {
		t.Fatalf("expected an unexpected file error, got: %v", err)
	}
}

func checkNamespaceKustomization(t *testing.T, fs *afero.Afero) {
	t.Helper()

	var got struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
			Labels      map[string]string `json:"labels"`
			Name        string            `json:"name"`
			Namespace   string            `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Decryption struct {
				SecretRef struct {
					Name string `json:"name"`
				} `json:"secretRef"`
			} `json:"decryption"`
			DependsOn []struct {
				Name string `json:"name"`
			} `json:"dependsOn"`
			Path      string `json:"path"`
			SourceRef struct {
				Name string `json:"name"`
			} `json:"sourceRef"`
			TargetNamespace string `json:"targetNamespace"`
		} `json:"spec"`
	}
	readInto(t, fs, testOrg+"/namespace.yaml", &got)

	summary := []string{
		got.Metadata.Name, got.Metadata.Namespace, got.Metadata.Labels["giantswarm.io/organization-namespace"],
		got.Metadata.Annotations["kustomize.toolkit.fluxcd.io/prune"],
		got.Spec.Decryption.SecretRef.Name, got.Spec.Path, got.Spec.SourceRef.Name, got.Spec.TargetNamespace,
	}
	for _, d := range got.Spec.DependsOn {
		summary = append(summary, d.Name)
	}
	expected := []string{
		"demomc-organization-demoorg", "default", "true", "disabled",
		"sops-gpg-master", "./management-clusters/demomc/organizations/demoorg/namespace", "gitops-demo", "org-demoorg",
		"demomc-gitops",
	}
	if diff := cmp.Diff(expected, summary); diff != "" {
		t.Errorf("namespace.yaml (-want +got):\n%s", diff)
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

// Test_Namespace_LeavesIgnoredKustomization checks that an organisation's
// kustomization.yaml that git ignores, which git cannot restore, is never
// edited: the upgrade refuses until it lists namespace.yaml, and then goes
// on, leaving it byte for byte.
func Test_Namespace_LeavesIgnoredKustomization(t *testing.T) {
	testCases := []struct {
		name      string
		existing  string
		expectErr bool
	}{
		{name: "without namespace.yaml", existing: "resources:\n  - demoorg.yaml\n  - workload-clusters\n", expectErr: true},
		{name: "listing namespace.yaml", existing: "resources:\n  - demoorg.yaml\n  - namespace.yaml\n  - workload-clusters\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := layoutRepo(t, map[string]string{
				testMC + "/demomc.yaml":                           namespaceMCKustomization,
				testOrg + "/demoorg.yaml":                         organization,
				testOrg + "/kustomization.yaml":                   tc.existing,
				testOrg + "/workload-clusters/kustomization.yaml": emptyResources,
			})
			layer := metadata.Layer{Kind: metadata.LayerOrganization, Path: testOrg, StructureVersion: 2}
			env := Env{Ignored: func(path string) bool { return path == testOrg+"/kustomization.yaml" }}

			changes, err := version3.Apply(fs, testRepo, layer, env)
			switch {
			case tc.expectErr && !IsUnexpectedFile(err):
				t.Fatalf("expected the ignored kustomization.yaml refused, got %v, %v", changes, err)
			case tc.expectErr && !strings.Contains(err.Error(), "git ignores"):
				t.Errorf("expected the error to say git ignores it, got: %s", err)
			case !tc.expectErr && err != nil:
				t.Fatalf("unexpected error: %s", err)
			}

			data, _ := fs.ReadFile(testRepo + "/" + testOrg + "/kustomization.yaml")
			if diff := cmp.Diff(tc.existing, string(data)); diff != "" {
				t.Errorf("expected the ignored kustomization.yaml kept byte for byte (-want +got):\n%s", diff)
			}
		})
	}
}
