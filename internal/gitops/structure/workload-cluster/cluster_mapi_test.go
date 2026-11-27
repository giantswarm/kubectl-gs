package wcluster

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// mapiSeed is a cluster added before structure version 3, whose
// Kustomization applies its `mapi` directory. Its organisation has the
// namespace Kustomization, so the refusal is down to `mapi` alone.
var mapiSeed = map[string]string{
	"/repo/management-clusters/demomc/organizations/demoorg/namespace.yaml":               "kind: Kustomization\n",
	"/repo/management-clusters/demomc/organizations/demoorg/namespace/kustomization.yaml": "resources: []\n",
	"/repo/" + wcsKustomizationFile:                    "resources:\n  - demowc.yaml\n",
	"/repo/" + wcFile:                                  "spec:\n  path: ./" + wcDir + "/mapi\n",
	"/repo/" + wcDir + "/mapi/apps/kustomization.yaml": "resources: []\n",
}

// Test_NewWorkloadCluster_RefusesMapiCluster checks that a cluster that still
// keeps its manifests in `mapi` is left alone: the cluster's Kustomization
// applies `mapi`, so Flux would never apply what a run adds next to it.
func Test_NewWorkloadCluster_RefusesMapiCluster(t *testing.T) {
	for _, base := range []string{"", testBase} {
		t.Run("base "+base, func(t *testing.T) {
			fs := seedFs(t, mapiSeed)

			err := createWorkloadCluster(t, fs, base)
			if !errors.Is(err, creator.ValidationError) {
				t.Fatalf("expected a validation error, got: %v", err)
			}
			for _, want := range []string{testWC, "mapi", "kubectl gs gitops upgrade"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected the error to mention %q, got: %s", want, err)
				}
			}

			checkUntouched(t, fs, mapiSeed, []string{wcKustomizationFile, appsDir, clusterDir, appsFile})
		})
	}
}

func seedFs(t *testing.T, seed map[string]string) *afero.Afero {
	t.Helper()

	fs := &afero.Afero{Fs: afero.NewMemMapFs()}
	for path, data := range seed {
		err := fs.WriteFile(path, []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	return fs
}

func createWorkloadCluster(t *testing.T, fs *afero.Afero, base string) error {
	t.Helper()

	config, err := NewWorkloadCluster(common.StructureConfig{
		ClusterBase:       base,
		ManagementCluster: testMC,
		Organization:      testOrg,
		Release:           testRelease,
		RepositoryName:    testRepository,
		WorkloadCluster:   testWC,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	config.Fs = fs
	config.Path = "/repo"
	config.Stdout = io.Discard

	return creator.NewCreator(*config).Create()
}

// checkUntouched fails when a seeded file changed, or one of absent, relative
// to the repository, was written.
func checkUntouched(t *testing.T, fs *afero.Afero, seed map[string]string, absent []string) {
	t.Helper()

	for _, path := range absent {
		exists, err := fs.Exists("/repo/" + path)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if exists {
			t.Errorf("expected nothing written, found %s", path)
		}
	}
	for path, data := range seed {
		got, err := fs.ReadFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if string(got) != data {
			t.Errorf("expected %s unchanged, got:\n%s", path, got)
		}
	}
}
