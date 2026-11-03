package wcluster

import (
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

func app(chart string) string {
	return appFromCatalog("cluster", chart)
}

func appFromCatalog(catalog, chart string) string {
	return `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: ${cluster_name}
spec:
  catalog: ` + catalog + `
  name: ` + chart + `
`
}

// releaseOverlay is the kustomization of gitops-template's
// bases/clusters/capa/release.
const releaseOverlay = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |-
      - op: replace
        path: /spec/name
        value: release-aws
      - op: replace
        path: /spec/version
        value: ${release}
    target:
      group: application.giantswarm.io
      kind: App
resources:
  - ../template
`

func Test_clusterChartName(t *testing.T) {
	testCases := []struct {
		name          string
		files         map[string]string
		expected      string
		expectInvalid bool
	}{
		{
			name: "cluster chart",
			files: map[string]string{
				"cluster.yaml":        app("cluster-aws"),
				"cluster_config.yaml": "global:\n  metadata:\n    name: ${cluster_name}\n",
			},
			expected: "cluster-aws",
		},
		{
			name: "release chart",
			files: map[string]string{
				"cluster.yaml": app("release-aws"),
			},
			expected: "release-aws",
		},
		{
			name: "other Apps are ignored",
			files: map[string]string{
				"apps.yaml":    app("hello-world") + "---\n" + app("cert-manager-app"),
				"cluster.yaml": app("cluster-azure"),
			},
			expected: "cluster-azure",
		},
		{
			name: "cluster chart from the test catalog",
			files: map[string]string{
				"cluster.yaml": appFromCatalog("cluster-test", "cluster-aws"),
			},
			expected: "cluster-aws",
		},
		{
			name: "cluster-* chart from another catalog is ignored",
			files: map[string]string{
				"apps.yaml":    appFromCatalog("default", "cluster-autoscaler"),
				"cluster.yaml": app("cluster-aws"),
			},
			expected: "cluster-aws",
		},
		{
			name: "overlay without App",
			files: map[string]string{
				"kustomization.yaml": "resources:\n  - ../../../../cluster_templates/hello_app_cluster\n",
			},
			expected: "",
		},
		{
			name: "overlay patching in a release chart",
			files: map[string]string{
				"kustomization.yaml": releaseOverlay,
			},
			expected: "release-aws",
		},
		{
			name: "overlay patching in a release chart from a patch file",
			files: map[string]string{
				"kustomization.yaml": "resources:\n  - ../template\npatches:\n  - path: patch_chart.yaml\n",
				"patch_chart.yaml":   "apiVersion: application.giantswarm.io/v1alpha1\nkind: App\nmetadata:\n  name: ${cluster_name}\nspec:\n  name: release-aws\n",
			},
			expected: "release-aws",
		},
		{
			name: "overlay patching in a cluster chart",
			files: map[string]string{
				"kustomization.yaml": strings.ReplaceAll(releaseOverlay, "release-aws", "cluster-aws"),
			},
			expected: "",
		},
		{
			name: "overlay patching another kind",
			files: map[string]string{
				"kustomization.yaml": strings.ReplaceAll(releaseOverlay, "kind: App", "kind: HelmRelease"),
			},
			expected: "",
		},
		{
			name: "no cluster App",
			files: map[string]string{
				"apps.yaml": app("hello-world"),
			},
			expected: "",
		},
		{
			name: "more than one cluster App",
			files: map[string]string{
				"a.yaml": app("cluster-aws"),
				"b.yaml": app("release-aws"),
			},
			expectInvalid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for name, data := range tc.files {
				err := afero.WriteFile(fs, "base/"+name, []byte(data), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			got, err := clusterChartName(fs, "base")
			if tc.expectInvalid {
				if microerror.Cause(err) != invalidConfigError {
					t.Fatalf("expected invalid config error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
