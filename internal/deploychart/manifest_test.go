package deploychart

import (
	"testing"
)

func TestBuildOCIRepository(t *testing.T) {
	tests := []struct {
		name     string
		opts     OCIRepositoryOptions
		expected string
	}{
		{
			name: "basic with pinned version",
			opts: OCIRepositoryOptions{
				Name:        "mycluster01-hello-world-app",
				Namespace:   "org-acme",
				ClusterName: "mycluster01",
				URL:         "oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app",
				Version:     "1.2.3",
				Interval:    "10m",
			},
			expected: `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  interval: 10m0s
  provider: generic
  ref:
    tag: 1.2.3
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
`,
		},
		{
			name: "no version",
			opts: OCIRepositoryOptions{
				Name:        "mycluster01-hello-world-app",
				Namespace:   "org-acme",
				ClusterName: "mycluster01",
				URL:         "oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app",
				Interval:    "10m",
			},
			expected: `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  interval: 10m0s
  provider: generic
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
`,
		},
		{
			name: "auto-upgrade patch",
			opts: OCIRepositoryOptions{
				Name:        "mycluster01-hello-world-app",
				Namespace:   "org-acme",
				ClusterName: "mycluster01",
				URL:         "oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app",
				Version:     "1.2.3",
				AutoUpgrade: "patch",
				Interval:    "10m",
			},
			expected: `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  interval: 10m0s
  provider: generic
  ref:
    semver: 1.2.x
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
`,
		},
		{
			name: "auto-upgrade minor",
			opts: OCIRepositoryOptions{
				Name:        "mycluster01-hello-world-app",
				Namespace:   "org-acme",
				ClusterName: "mycluster01",
				URL:         "oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app",
				Version:     "1.2.3",
				AutoUpgrade: "minor",
				Interval:    "10m",
			},
			expected: `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  interval: 10m0s
  provider: generic
  ref:
    semver: 1.x
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
`,
		},
		{
			name: "auto-upgrade all",
			opts: OCIRepositoryOptions{
				Name:        "mycluster01-hello-world-app",
				Namespace:   "org-acme",
				ClusterName: "mycluster01",
				URL:         "oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app",
				Version:     "1.2.3",
				AutoUpgrade: "all",
				Interval:    "10m",
			},
			expected: `apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  interval: 10m0s
  provider: generic
  ref:
    semver: '*'
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world-app
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildOCIRepository(tc.opts)
			got, err := MarshalManifest(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.expected {
				t.Errorf("unexpected output:\n--- got ---\n%s\n--- expected ---\n%s", string(got), tc.expected)
			}
		})
	}
}

func TestBuildHelmRelease(t *testing.T) {
	tests := []struct {
		name     string
		opts     HelmReleaseOptions
		expected string
	}{
		{
			name: "basic workload cluster",
			opts: HelmReleaseOptions{
				Name:            "mycluster01-hello-world-app",
				Namespace:       "org-acme",
				ClusterName:     "mycluster01",
				ChartName:       "hello-world-app",
				TargetNamespace: "hello",
				Interval:        "10m",
			},
			expected: `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  chartRef:
    kind: OCIRepository
    name: mycluster01-hello-world-app
  install:
    createNamespace: true
  interval: 10m0s
  kubeConfig:
    secretRef:
      name: mycluster01-kubeconfig
  releaseName: hello-world-app
  targetNamespace: hello
`,
		},
		{
			name: "with values",
			opts: HelmReleaseOptions{
				Name:            "mycluster01-hello-world-app",
				Namespace:       "org-acme",
				ClusterName:     "mycluster01",
				ChartName:       "hello-world-app",
				TargetNamespace: "hello",
				Interval:        "10m",
				Values: map[string]any{
					"replicaCount": 3,
					"ingress": map[string]any{
						"enabled": true,
						"host":    "hello.example.com",
					},
				},
			},
			expected: `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  labels:
    giantswarm.io/cluster: mycluster01
  name: mycluster01-hello-world-app
  namespace: org-acme
spec:
  chartRef:
    kind: OCIRepository
    name: mycluster01-hello-world-app
  install:
    createNamespace: true
  interval: 10m0s
  kubeConfig:
    secretRef:
      name: mycluster01-kubeconfig
  releaseName: hello-world-app
  targetNamespace: hello
  values:
    ingress:
      enabled: true
      host: hello.example.com
    replicaCount: 3
`,
		},
		{
			name: "management cluster mode",
			opts: HelmReleaseOptions{
				Name:              "mymc01-hello-world-app",
				Namespace:         "org-acme",
				ClusterName:       "mymc01",
				ChartName:         "hello-world-app",
				TargetNamespace:   "hello",
				Interval:          "10m",
				ManagementCluster: true,
			},
			expected: `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  labels:
    giantswarm.io/cluster: mymc01
  name: mymc01-hello-world-app
  namespace: org-acme
spec:
  chartRef:
    kind: OCIRepository
    name: mymc01-hello-world-app
  install:
    createNamespace: true
  interval: 10m0s
  releaseName: hello-world-app
  targetNamespace: hello
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildHelmRelease(tc.opts)
			got, err := MarshalManifest(result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.expected {
				t.Errorf("unexpected output:\n--- got ---\n%s\n--- expected ---\n%s", string(got), tc.expected)
			}
		})
	}
}

func TestSemverRange(t *testing.T) {
	tests := []struct {
		version     string
		autoUpgrade string
		expected    string
	}{
		{"1.2.3", "patch", "1.2.x"},
		{"1.2.3", "minor", "1.x"},
		{"1.2.3", "all", "*"},
		{"0.1.0", "patch", "0.1.x"},
		{"0.1.0", "minor", "0.x"},
		{"2.0.0", "patch", "2.0.x"},
		{"v1.2.3", "patch", "1.2.x"},
		{"v0.5.0", "minor", "0.x"},
	}

	for _, tc := range tests {
		t.Run(tc.version+"_"+tc.autoUpgrade, func(t *testing.T) {
			got := SemverRange(tc.version, tc.autoUpgrade)
			if got != tc.expected {
				t.Errorf("SemverRange(%q, %q) = %q, want %q", tc.version, tc.autoUpgrade, got, tc.expected)
			}
		})
	}
}

