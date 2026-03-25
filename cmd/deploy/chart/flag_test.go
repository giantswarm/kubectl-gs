package chart

import (
	"testing"
)

func TestFlagValidate(t *testing.T) {
	tests := []struct {
		name    string
		flag    flag
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid basic flags",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: "oci://gsoci.azurecr.io/charts/giantswarm/",
				Interval:     "10m",
			},
		},
		{
			name: "missing chart name",
			flag: flag{
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
			},
			wantErr: true,
			errMsg:  "chart-name",
		},
		{
			name: "chart name with slash",
			flag: flag{
				ChartName:    "charts/hello",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
			},
			wantErr: true,
			errMsg:  "must not contain '/'",
		},
		{
			name: "missing organization",
			flag: flag{
				ChartName:    "hello-world-app",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
			},
			wantErr: true,
			errMsg:  "organization",
		},
		{
			name: "missing cluster",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
			},
			wantErr: true,
			errMsg:  "target-cluster",
		},
		{
			name: "missing target namespace",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
			},
			wantErr: true,
			errMsg:  "target-namespace",
		},
		{
			name: "invalid auto-upgrade value",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				AutoUpgrade:  "invalid",
				Version:      "1.0.0",
			},
			wantErr: true,
			errMsg:  "auto-upgrade",
		},
		{
			name: "auto-upgrade without version is valid (resolved at runtime)",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				AutoUpgrade:  "patch",
			},
		},
		{
			name: "valid auto-upgrade patch",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				AutoUpgrade:  "patch",
				Version:      "1.2.3",
			},
		},
		{
			name: "auto-upgrade all without version is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				AutoUpgrade:  "all",
			},
		},
		{
			name: "invalid version format",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				Version:      "foo",
			},
			wantErr: true,
			errMsg:  "valid semver",
		},
		{
			name: "version with v prefix is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				Version:      "v1.2.3",
			},
		},
		{
			name: "oci prefix normalized",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: "example.com/charts",
				Interval:     defaultInterval,
			},
		},
		{
			name: "management-cluster without target-cluster is valid",
			flag: flag{
				ChartName:         "hello-world-app",
				Organization:      "acme",
				TargetNS:          "hello",
				OCIURLPrefix:      defaultOCIURLPrefix,
				Interval:          defaultInterval,
				ManagementCluster: true,
			},
		},
		{
			name: "management-cluster with target-cluster is invalid",
			flag: flag{
				ChartName:         "hello-world-app",
				Organization:      "acme",
				Cluster:           "mycluster01",
				TargetNS:          "hello",
				OCIURLPrefix:      defaultOCIURLPrefix,
				Interval:          defaultInterval,
				ManagementCluster: true,
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "dry-run flag is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				DryRun:       true,
			},
		},
		{
			name: "values-from ConfigMap is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"ConfigMap/my-config"},
			},
		},
		{
			name: "values-from Secret is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"Secret/my-secret"},
			},
		},
		{
			name: "values-from lowercase configmap is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"configmap/my-config"},
			},
		},
		{
			name: "values-from lowercase secret is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"secret/my-secret"},
			},
		},
		{
			name: "values-from multiple entries is valid",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"ConfigMap/config1", "Secret/secret1"},
			},
		},
		{
			name: "values-from invalid format",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"my-config"},
			},
			wantErr: true,
			errMsg:  "ConfigMap/name or Secret/name",
		},
		{
			name: "registry-provider aws is valid",
			flag: flag{
				ChartName:        "hello-world-app",
				Organization:     "acme",
				Cluster:          "mycluster01",
				TargetNS:         "hello",
				OCIURLPrefix:     defaultOCIURLPrefix,
				Interval:         defaultInterval,
				RegistryProvider: "aws",
			},
		},
		{
			name: "registry-provider invalid value",
			flag: flag{
				ChartName:        "hello-world-app",
				Organization:     "acme",
				Cluster:          "mycluster01",
				TargetNS:         "hello",
				OCIURLPrefix:     defaultOCIURLPrefix,
				Interval:         defaultInterval,
				RegistryProvider: "invalid",
			},
			wantErr: true,
			errMsg:  "registry-provider",
		},
		{
			name: "registry-provider and registry-username mutually exclusive",
			flag: flag{
				ChartName:        "hello-world-app",
				Organization:     "acme",
				Cluster:          "mycluster01",
				TargetNS:         "hello",
				OCIURLPrefix:     defaultOCIURLPrefix,
				Interval:         defaultInterval,
				RegistryProvider: "aws",
				RegistryUsername: "myuser",
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "values-from invalid kind",
			flag: flag{
				ChartName:    "hello-world-app",
				Organization: "acme",
				Cluster:      "mycluster01",
				TargetNS:     "hello",
				OCIURLPrefix: defaultOCIURLPrefix,
				Interval:     defaultInterval,
				ValuesFrom:   []string{"Deployment/my-deploy"},
			},
			wantErr: true,
			errMsg:  "must be ConfigMap or Secret",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.flag.Validate()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errMsg != "" {
					if got := err.Error(); !contains(got, tc.errMsg) {
						t.Errorf("error %q should contain %q", got, tc.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNormalizeOCIURLPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"oci://example.com/charts/", "oci://example.com/charts/"},
		{"example.com/charts", "oci://example.com/charts/"},
		{"oci://example.com/charts", "oci://example.com/charts/"},
		{"example.com/charts/", "oci://example.com/charts/"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeOCIURLPrefix(tc.input)
			if got != tc.expected {
				t.Errorf("normalizeOCIURLPrefix(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
