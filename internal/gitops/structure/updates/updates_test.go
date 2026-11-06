package updates

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	appDir        = "management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/mapi/apps/hello-world"
	ociRepository = `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: demowc-hello-world
  namespace: org-demoorg
spec:
  interval: 10m0s
  ref:
    tag: 0.3.0
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world
`
)

func Test_NewAutomaticUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		files        map[string]string
		expected     string
		expectErrMsg string
	}{
		{
			name:     "OCIRepository",
			files:    map[string]string{"oci-repository.yaml": ociRepository},
			expected: strings.Replace(ociRepository, "tag: 0.3.0", "semver: '>=1.0.0'", 1),
		},
		{
			name:         "App CR",
			files:        map[string]string{"appcr.yaml": "apiVersion: application.giantswarm.io/v1alpha1\nkind: App\n"},
			expectErrMsg: "no longer supports",
		},
		{
			name:         "app from a base",
			files:        map[string]string{"kustomization.yaml": "resources:\n  - ../../../../../../../../../bases/apps/hello-world\n"},
			expectErrMsg: "has no `oci-repository.yaml` file",
		},
		{
			name:         "missing app",
			expectErrMsg: "has no `oci-repository.yaml` file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := fs.MkdirAll("/repo", 0755)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			for name, data := range tc.files {
				err := afero.WriteFile(fs, filepath.Join("/repo", appDir, name), []byte(data), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
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
			if tc.expectErrMsg != "" {
				if microerror.Cause(err) != creator.ValidationError || !strings.Contains(err.Error(), tc.expectErrMsg) {
					t.Fatalf("expected a validation error containing %q, got: %v", tc.expectErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			got, err := afero.ReadFile(fs, filepath.Join("/repo", appDir, "oci-repository.yaml"))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected oci-repository.yaml (-want +got):\n%s", diff)
			}
		})
	}
}
