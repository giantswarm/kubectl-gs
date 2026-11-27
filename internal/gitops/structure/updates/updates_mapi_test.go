package updates

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

// Test_NewAutomaticUpdate_RefusesMapiCluster checks that automatic updates
// are not configured for a cluster that still keeps its manifests in `mapi`.
func Test_NewAutomaticUpdate_RefusesMapiCluster(t *testing.T) {
	fs := afero.NewMemMapFs()
	mapiApp := filepath.Join("/repo", filepath.Dir(filepath.Dir(appDir)), "mapi", "apps", "hello-world", "oci-repository.yaml")
	err := afero.WriteFile(fs, mapiApp, []byte(ociRepository), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	config, err := NewAutomaticUpdate(common.StructureConfig{
		AppName:           "hello-world",
		AppVersionRange:   ">=1.0.0",
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
}
