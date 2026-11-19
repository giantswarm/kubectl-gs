package common

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildClusterFluxResources(t *testing.T) {
	config := ClusterConfig{
		Name:           "e9sc4",
		Organization:   "laszlo",
		ReleaseVersion: "35.0.1",
	}

	ociRepoYAML, helmReleaseYAML, err := BuildClusterFluxResources(config, "release-aws", "e9sc4-userconfig")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expectedOCIRepo := `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: e9sc4
  name: e9sc4
  namespace: org-laszlo
spec:
  interval: 10m0s
  provider: generic
  ref:
    tag: 35.0.1
  timeout: 1m0s
  url: oci://gsoci.azurecr.io/charts/giantswarm/release-aws
`
	if diff := cmp.Diff(strings.Split(expectedOCIRepo, "\n"), strings.Split(string(ociRepoYAML), "\n")); diff != "" {
		t.Errorf("unexpected OCIRepository (-want +got):\n%s", diff)
	}

	expectedHelmRelease := `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  labels:
    giantswarm.io/cluster: e9sc4
  name: e9sc4
  namespace: org-laszlo
spec:
  chartRef:
    kind: OCIRepository
    name: e9sc4
    namespace: org-laszlo
  install:
    remediation:
      remediateLastFailure: false
  interval: 5m0s
  releaseName: e9sc4
  serviceAccountName: automation
  storageNamespace: org-laszlo
  targetNamespace: org-laszlo
  timeout: 10m0s
  upgrade:
    remediation:
      remediateLastFailure: true
      retries: 10
      strategy: rollback
  valuesFrom:
  - kind: ConfigMap
    name: e9sc4-userconfig
    valuesKey: values
`
	if diff := cmp.Diff(strings.Split(expectedHelmRelease, "\n"), strings.Split(string(helmReleaseYAML), "\n")); diff != "" {
		t.Errorf("unexpected HelmRelease (-want +got):\n%s", diff)
	}
}

func TestIsReleaseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "release version 35.0.0 returns true",
			version:  "35.0.0",
			expected: true,
		},
		{
			name:     "release version v35.0.0 with v prefix returns true",
			version:  "v35.0.0",
			expected: true,
		},
		{
			name:     "release version 36.1.2 returns true",
			version:  "36.1.2",
			expected: true,
		},
		{
			name:     "chart version 7.2.5 returns false",
			version:  "7.2.5",
			expected: false,
		},
		{
			name:     "chart version 1.0.0 returns false",
			version:  "1.0.0",
			expected: false,
		},
		{
			name:     "chart version 34.9.9 returns false",
			version:  "34.9.9",
			expected: false,
		},
		{
			name:     "empty version returns false",
			version:  "",
			expected: false,
		},
		{
			name:     "invalid version returns false",
			version:  "invalid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReleaseVersion(tt.version)
			if result != tt.expected {
				t.Errorf("IsReleaseVersion(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestClusterResourceLabels(t *testing.T) {
	testCases := []struct {
		name   string
		config ClusterConfig
		want   map[string]string
	}{
		{
			name:   "no labels",
			config: ClusterConfig{},
			want:   map[string]string{},
		},
		{
			name:   "deletion prevention",
			config: ClusterConfig{PreventDeletion: true},
			want:   map[string]string{"giantswarm.io/prevent-deletion": "true"},
		},
		{
			name: "user labels, without the cluster label",
			config: ClusterConfig{
				PreventDeletion: true,
				Labels: map[string]string{
					"environment":           "testing",
					"giantswarm.io/cluster": "another-cluster",
				},
			},
			want: map[string]string{
				"environment":                    "testing",
				"giantswarm.io/prevent-deletion": "true",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, clusterResourceLabels(tc.config)); diff != "" {
				t.Errorf("unexpected labels (-want +got):\n%s", diff)
			}
		})
	}
}

// TestBuildClusterFluxResourcesKeepsClusterLabel checks that a user label
// can't replace the giantswarm.io/cluster label of the Flux resources.
func TestBuildClusterFluxResourcesKeepsClusterLabel(t *testing.T) {
	config := ClusterConfig{
		Name:           "e9sc4",
		Organization:   "laszlo",
		ReleaseVersion: "35.0.1",
		Labels:         map[string]string{"giantswarm.io/cluster": "another-cluster"},
	}

	ociRepoYAML, helmReleaseYAML, err := BuildClusterFluxResources(config, "release-aws", "e9sc4-userconfig")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for kind, manifest := range map[string][]byte{"OCIRepository": ociRepoYAML, "HelmRelease": helmReleaseYAML} {
		if !strings.Contains(string(manifest), "giantswarm.io/cluster: e9sc4\n") || strings.Contains(string(manifest), "another-cluster") {
			t.Errorf("%s doesn't keep the cluster label:\n%s", kind, manifest)
		}
	}
}
