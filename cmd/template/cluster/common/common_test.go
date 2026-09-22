package common

import "testing"

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
	if string(ociRepoYAML) != expectedOCIRepo {
		t.Errorf("unexpected OCIRepository:\n--- got ---\n%s\n--- expected ---\n%s", ociRepoYAML, expectedOCIRepo)
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
      retries: 10
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
	if string(helmReleaseYAML) != expectedHelmRelease {
		t.Errorf("unexpected HelmRelease:\n--- got ---\n%s\n--- expected ---\n%s", helmReleaseYAML, expectedHelmRelease)
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
