package metadata

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

// Test_Adopt_RecordsTheLayoutVersion checks that a workload cluster is
// recorded at the structure version its layout has: one with the `mapi`
// directory, or without the `kustomization.yaml` that keeps `out-of-band` from
// the management cluster, is at version 2, with its apps, so that
// `gitops upgrade` moves it.

// Test_Adopt_OrganizationWithoutNamespace checks that an organisation
// without its namespace Kustomization is recorded at version 2, so that
// `gitops upgrade` adds it.

const (
	adoptMC  = "management-clusters/demomc"
	adoptOrg = adoptMC + "/organizations/demoorg"

	// keyFile stands in for a SOPS key Secret, and mcKeys is where the
	// management cluster keeps them.
	keyFile = "kind: Secret\n"
	mcKeys  = "management-clusters/mymc/" + "secrets"
)

func Test_Adopt_ManagementClusterSecrets(t *testing.T) {
	testCases := []struct {
		name     string
		files    map[string]string
		expected int
	}{
		{
			name: "keys without a kustomization.yaml",
			files: map[string]string{
				mcKeys + "/mymc.gpgkey.enc.yaml": keyFile,
				mcKeys + "/wc1.gpgkey.enc.yaml":  keyFile,
			},
			expected: lastMapiVersion,
		},
		{
			name: "keys with a kustomization.yaml",
			files: map[string]string{
				mcKeys + "/mymc.gpgkey.enc.yaml": keyFile,
				mcKeys + "/kustomization.yaml":   "resources:\n  - mymc.gpgkey.enc.yaml\n",
			},
			expected: StructureVersion,
		},
		{
			name:     "no secrets",
			files:    map[string]string{"management-clusters/mymc/mymc.yaml": "kind: Kustomization\n"},
			expected: StructureVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := adoptedManagementCluster(t, tc.files); got != tc.expected {
				t.Errorf("expected the management cluster at version %d, got %d", tc.expected, got)
			}
		})
	}
}

// adoptedManagementCluster adopts a repository of the files, and returns the
// version it records the management cluster at.
func adoptedManagementCluster(t *testing.T, files map[string]string) int {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	for path, data := range files {
		err := fs.WriteFile(filepath.Join(testRepo, path), []byte(data), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, l := range result.Metadata.Layers {
		if l.Kind == LayerManagementCluster {
			return l.StructureVersion
		}
	}

	return 0
}

func Test_Adopt_RecordsTheLayoutVersion(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}

	const wcsDir = "management-clusters/demomc/organizations/demoorg/workload-clusters"
	helmRelease := "apiVersion: helm.toolkit.fluxcd.io/v2\nkind: HelmRelease\n"
	files := map[string]string{
		wcsDir + "/mapiwc/mapi/cluster/kustomization.yaml":          emptyResources,
		wcsDir + "/mapiwc/mapi/apps/hello-world/helm-release.yaml":  helmRelease,
		wcsDir + "/flatwc/cluster/kustomization.yaml":               emptyResources,
		wcsDir + "/flatwc/apps/hello-world/helm-release.yaml":       helmRelease,
		wcsDir + "/currentwc/kustomization.yaml":                    "resources:\n  - apps/\n",
		wcsDir + "/currentwc/apps/hello-world/helm-release.yaml":    helmRelease,
		wcsDir + "/currentwc/out-of-band/configmaps/configmap.yaml": "kind: ConfigMap\n",
	}
	for path, data := range files {
		err := fs.WriteFile(filepath.Join(testRepo, path), []byte(data), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := map[string]int{}
	for _, l := range result.Metadata.Layers {
		got[l.Path] = l.StructureVersion
	}
	expected := map[string]int{
		adoptMC:                                  StructureVersion,
		adoptOrg:                                 lastMapiVersion,
		wcsDir + "/mapiwc":                       2,
		wcsDir + "/mapiwc/mapi/apps/hello-world": 2,
		wcsDir + "/flatwc":                       2,
		wcsDir + "/flatwc/apps/hello-world":      StructureVersion,
		wcsDir + "/currentwc":                    StructureVersion,
		wcsDir + "/currentwc/apps/hello-world":   StructureVersion,
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("unexpected versions (-want +got):\n%s", diff)
	}
}

func Test_Adopt_OrganizationWithoutNamespace(t *testing.T) {
	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	files := map[string]string{
		"management-clusters/demomc/organizations/demoorg/demoorg.yaml":                "kind: Organization\n",
		"management-clusters/demomc/organizations/neworg/neworg.yaml":                  "kind: Organization\n",
		"management-clusters/demomc/organizations/neworg/namespace.yaml":               "kind: Kustomization\n",
		"management-clusters/demomc/organizations/neworg/namespace/kustomization.yaml": emptyResources,
	}
	for path, data := range files {
		err := fs.WriteFile(filepath.Join(testRepo, path), []byte(data), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	result, err := Adopt(fs, testRepo)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := map[string]int{}
	for _, l := range result.Metadata.Layers {
		got[l.Path] = l.StructureVersion
	}
	expected := map[string]int{
		adoptMC:  StructureVersion,
		adoptOrg: lastMapiVersion,
		"management-clusters/demomc/organizations/neworg": StructureVersion,
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("unexpected versions (-want +got):\n%s", diff)
	}
}
