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
