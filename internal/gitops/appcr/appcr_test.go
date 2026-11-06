package appcr

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

const (
	app = `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: hello
`
	helmRelease = `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: hello
`
)

func Test_Files(t *testing.T) {
	testCases := []struct {
		name     string
		files    map[string]string
		dir      string
		expected []string
	}{
		{
			name: "no kustomization counts every YAML file",
			files: map[string]string{
				"apps/hello/appcr.yaml":     app,
				"apps/hello/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n",
			},
			dir:      "apps/hello",
			expected: []string{"apps/hello/appcr.yaml"},
		},
		{
			name: "no kustomization counts subdirectories",
			files: map[string]string{
				"apps/hello/appcr.yaml":        app,
				"apps/world/helm-release.yaml": helmRelease,
			},
			dir:      "apps",
			expected: []string{"apps/hello/appcr.yaml"},
		},
		{
			name: "patches are not resources",
			files: map[string]string{
				"apps/kustomization.yaml":         "patches:\n  - path: patch_cluster_config.yaml\nresources:\n  - hello/helm-release.yaml\n",
				"apps/patch_cluster_config.yaml":  app,
				"apps/hello/helm-release.yaml":    helmRelease,
				"apps/unlisted/appcr.yaml":        app,
				"apps/unlisted/kustomization.yml": "",
			},
			dir:      "apps",
			expected: []string{},
		},
		{
			name: "follows directories into bases",
			files: map[string]string{
				"wc/apps/kustomization.yaml":       "resources:\n  - hello\n  - github.com/example/remote\n",
				"wc/apps/hello/kustomization.yaml": "resources:\n  - ../../../bases/hello\n",
				"bases/hello/kustomization.yaml":   "resources:\n  - app.yaml\n",
				"bases/hello/app.yaml":             helmRelease + "---\n" + app,
			},
			dir:      "wc/apps",
			expected: []string{"bases/hello/app.yaml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for name, data := range tc.files {
				err := afero.WriteFile(fs, name, []byte(data), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			got, err := Files(fs, tc.dir)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("unexpected files (-want +got):\n%s", diff)
			}
		})
	}
}
