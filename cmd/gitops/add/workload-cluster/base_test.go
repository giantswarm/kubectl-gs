package wcluster

import (
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

func ociRepository(chart string) string {
	return `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: ${cluster_name}
spec:
  url: oci://gsoci.azurecr.io/charts/giantswarm/` + chart + `
`
}

func Test_clusterChartName(t *testing.T) {
	testCases := []struct {
		name          string
		files         map[string]string
		expected      string
		expectInvalid bool
	}{
		{
			name: "release chart",
			files: map[string]string{
				"cluster.yaml":        ociRepository("release-aws"),
				"cluster_config.yaml": "global:\n  metadata:\n    name: ${cluster_name}\n",
			},
			expected: "release-aws",
		},
		{
			name: "other OCIRepositories and Apps are ignored",
			files: map[string]string{
				"apps.yaml":    ociRepository("hello-world") + "---\n" + appFromCatalog("default", "cluster-autoscaler"),
				"cluster.yaml": ociRepository("release-azure"),
			},
			expected: "release-azure",
		},
		{
			name: "overlay without release chart",
			files: map[string]string{
				"kustomization.yaml": "resources:\n  - ../../../../cluster_templates/hello_app_cluster\n",
			},
			expected: "",
		},
		{
			name: "JSON 6902 patch file",
			files: map[string]string{
				"cluster.yaml": ociRepository("release-aws"),
				"ops.yaml":     "- op: replace\n  path: /spec/name\n  value: release-aws\n",
			},
			expected: "release-aws",
		},
		{
			name: "overlay of a cluster App patching in a release chart",
			files: map[string]string{
				"kustomization.yaml":             releaseOverlay,
				"../template/kustomization.yaml": "resources:\n  - cluster.yaml\n",
				"../template/cluster.yaml":       app("cluster-aws"),
			},
			expectInvalid: true,
		},
		{
			name: "cluster App",
			files: map[string]string{
				"cluster.yaml": app("cluster-aws"),
			},
			expectInvalid: true,
		},
		{
			name: "release chart App from the test catalog",
			files: map[string]string{
				"cluster.yaml": appFromCatalog("cluster-test", "release-aws"),
			},
			expectInvalid: true,
		},
		{
			name: "more than one release chart",
			files: map[string]string{
				"a.yaml": ociRepository("release-aws"),
				"b.yaml": ociRepository("release-azure"),
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
