package app

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// Test_NewApp_RefusesMapiCluster checks that an app is not added to a cluster
// that still keeps its manifests in `mapi`, as Flux would never apply it.
func Test_NewApp_RefusesMapiCluster(t *testing.T) {
	wcDir := "/repo/management-clusters/demomc/organizations/demoorg/workload-clusters/demowc"
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, wcDir+"/mapi/apps/kustomization.yaml", []byte("resources: []\n"), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	config, err := NewApp(common.StructureConfig{
		App:               "hello-world",
		AppName:           "hello-world",
		AppOCIURLPrefix:   "oci://gsoci.azurecr.io/charts/giantswarm",
		AppVersion:        "0.3.0",
		ManagementCluster: "demomc",
		Organization:      "demoorg",
		WorkloadCluster:   "demowc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	config.Fs = fs
	config.Path = "/repo"
	config.Stdout = io.Discard

	err = creator.NewCreator(*config).Create()
	if !errors.Is(err, creator.ValidationError) {
		t.Fatalf("expected a validation error, got: %v", err)
	}
	for _, want := range []string{"demowc", "mapi", "kubectl gs gitops upgrade"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected the error to mention %q, got: %s", want, err)
		}
	}

	exists, err := afero.DirExists(fs, wcDir+"/apps")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if exists {
		t.Errorf("expected nothing written outside mapi, found %s/apps", wcDir)
	}
}
