package chart

import (
	"testing"
)

func TestSplitOCIURLPrefix(t *testing.T) {
	tests := []struct {
		name         string
		ociURLPrefix string
		chartName    string
		wantRegistry string
		wantRepoPath string
	}{
		{
			name:         "default Giant Swarm registry",
			ociURLPrefix: "oci://gsoci.azurecr.io/charts/giantswarm/",
			chartName:    "hello-world",
			wantRegistry: "gsoci.azurecr.io",
			wantRepoPath: "charts/giantswarm/hello-world",
		},
		{
			name:         "custom registry with path",
			ociURLPrefix: "oci://registry.example.com/charts/",
			chartName:    "my-app",
			wantRegistry: "registry.example.com",
			wantRepoPath: "charts/my-app",
		},
		{
			name:         "registry without path prefix",
			ociURLPrefix: "oci://registry.example.com/",
			chartName:    "my-app",
			wantRegistry: "registry.example.com",
			wantRepoPath: "my-app",
		},
		{
			name:         "deeply nested path",
			ociURLPrefix: "oci://registry.example.com/org/team/charts/",
			chartName:    "my-app",
			wantRegistry: "registry.example.com",
			wantRepoPath: "org/team/charts/my-app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry, repoPath := splitOCIURLPrefix(tc.ociURLPrefix, tc.chartName)
			if registry != tc.wantRegistry {
				t.Errorf("registry = %q, want %q", registry, tc.wantRegistry)
			}
			if repoPath != tc.wantRepoPath {
				t.Errorf("repoPath = %q, want %q", repoPath, tc.wantRepoPath)
			}
		})
	}
}
