package wcluster

import (
	"context"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

const testRelease = `apiVersion: release.giantswarm.io/v1alpha1
kind: Release
metadata:
  name: aws-36.1.1
spec:
  apps:
  - name: aws-ebs-csi-driver
    version: 4.3.2
  components:
  - name: cluster-aws
    catalog: cluster
    version: 11.0.2
  - name: kubernetes
    version: 1.36.5
  date: "2026-10-25T16:28:48Z"
  state: active
`

const testBaseApp = `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: ${cluster_name}
  namespace: org-${organization}
spec:
  catalog: cluster
  name: cluster-aws
  namespace: org-${organization}
  version: ""
`

const testBaseKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - cluster.yaml
`

func testBaseFs(t *testing.T, files map[string]string) afero.Fs {
	t.Helper()

	fs := afero.NewMemMapFs()
	for name, content := range files {
		err := afero.WriteFile(fs, "/repo/bases/clusters/capa/template/"+name, []byte(content), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	return fs
}

func fakeFetch(t *testing.T, wantURL string, body string, err error) fetchFunc {
	return func(ctx context.Context, url string) ([]byte, error) {
		if url != wantURL {
			t.Fatalf("expected fetch of %s, got %s", wantURL, url)
		}
		return []byte(body), err
	}
}

func Test_clusterChartName(t *testing.T) {
	testCases := []struct {
		name          string
		files         map[string]string
		expected      string
		expectInvalid bool
	}{
		{
			name: "app in base",
			files: map[string]string{
				"kustomization.yaml":  testBaseKustomization,
				"cluster.yaml":        testBaseApp,
				"cluster_config.yaml": "global:\n  metadata:\n    name: ${cluster_name}\n",
				"readme.md":           "# CAPA template\n",
			},
			expected: "cluster-aws",
		},
		{
			name: "app in a multi-document file",
			files: map[string]string{
				"resources.yaml": testBaseKustomization + "---\n" + testBaseApp,
			},
			expected: "cluster-aws",
		},
		{
			name: "no app",
			files: map[string]string{
				"kustomization.yaml": testBaseKustomization,
			},
			expectInvalid: true,
		},
		{
			name: "app without name",
			files: map[string]string{
				"cluster.yaml": strings.Replace(testBaseApp, "  name: cluster-aws\n", "", 1),
			},
			expectInvalid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, err := clusterChartName(testBaseFs(t, tc.files), "/repo/bases/clusters/capa/template")
			if tc.expectInvalid {
				if microerror.Cause(err) != invalidBaseError {
					t.Fatalf("expected invalid base error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if name != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, name)
			}
		})
	}
}

func Test_releaseDirectory(t *testing.T) {
	testCases := map[string]string{
		"cluster-aws":            "capa",
		"cluster-azure":          "azure",
		"cluster-vsphere":        "vsphere",
		"cluster-cloud-director": "cloud-director",
		"cluster-eks":            "eks",
	}

	for chart, expected := range testCases {
		if dir := releaseDirectory(chart); dir != expected {
			t.Errorf("expected %q for %s, got %q", expected, chart, dir)
		}
	}
}

func Test_releaseClusterAppVersion(t *testing.T) {
	const url = "https://raw.githubusercontent.com/giantswarm/releases/master/capa/v36.1.1/release.yaml"

	testCases := []struct {
		name          string
		release       string
		chartName     string
		fetch         func(t *testing.T) fetchFunc
		expected      string
		expectedError string
	}{
		{
			name:      "found",
			release:   "36.1.1",
			chartName: "cluster-aws",
			fetch:     func(t *testing.T) fetchFunc { return fakeFetch(t, url, testRelease, nil) },
			expected:  "11.0.2",
		},
		{
			name:      "found with v prefix",
			release:   "v36.1.1",
			chartName: "cluster-aws",
			fetch:     func(t *testing.T) fetchFunc { return fakeFetch(t, url, testRelease, nil) },
			expected:  "11.0.2",
		},
		{
			name:      "release not found",
			release:   "36.1.1",
			chartName: "cluster-aws",
			fetch: func(t *testing.T) fetchFunc {
				return fakeFetch(t, url, "", microerror.Maskf(notFoundError, "%s was not found", url))
			},
			expectedError: "release v36.1.1 was not found for provider capa",
		},
		{
			name:      "component missing",
			release:   "36.1.1",
			chartName: "cluster-aws",
			fetch: func(t *testing.T) fetchFunc {
				return fakeFetch(t, url, strings.Replace(testRelease, "cluster-aws", "cluster-other", 1), nil)
			},
			expectedError: "release v36.1.1 for provider capa has no cluster-aws component",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, err := releaseClusterAppVersion(context.Background(), tc.fetch(t), tc.chartName, tc.release)
			if tc.expectedError != "" {
				if !IsNotFound(err) || !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected not found error with %q, got: %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if version != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, version)
			}
		})
	}
}

func Test_runner_clusterAppVersion(t *testing.T) {
	fs := testBaseFs(t, map[string]string{
		"kustomization.yaml": testBaseKustomization,
		"cluster.yaml":       testBaseApp,
	})

	t.Run("from release", func(t *testing.T) {
		r := &runner{
			flag: &flag{Base: "bases/clusters/capa/template", Release: "36.1.1"},
			fs:   fs,
			fetch: fakeFetch(t,
				"https://raw.githubusercontent.com/giantswarm/releases/master/capa/v36.1.1/release.yaml",
				testRelease, nil),
		}

		version, err := r.clusterAppVersion(context.Background(), "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if version != "11.0.2" {
			t.Fatalf("expected 11.0.2, got %q", version)
		}
	})

	t.Run("override skips the fetch", func(t *testing.T) {
		r := &runner{
			flag: &flag{Base: "bases/clusters/capa/template", Release: "36.1.1", ClusterAppVersion: "10.0.0"},
			fs:   fs,
			fetch: func(ctx context.Context, url string) ([]byte, error) {
				t.Fatalf("unexpected fetch of %s", url)
				return nil, nil
			},
		}

		version, err := r.clusterAppVersion(context.Background(), "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if version != "10.0.0" {
			t.Fatalf("expected 10.0.0, got %q", version)
		}
	})
}
