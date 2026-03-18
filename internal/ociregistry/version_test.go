package ociregistry

import (
	"testing"
)

func TestLatestSemverTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
		wantErr  bool
	}{
		{
			name:     "basic semver tags",
			tags:     []string{"1.0.0", "1.1.0", "1.2.3", "0.9.0"},
			expected: "1.2.3",
		},
		{
			name:     "with v prefix",
			tags:     []string{"v1.0.0", "v2.0.0", "v1.5.0"},
			expected: "v2.0.0",
		},
		{
			name:     "mixed with non-semver tags",
			tags:     []string{"latest", "1.0.0", "main", "2.1.0", "sha-abc123"},
			expected: "2.1.0",
		},
		{
			name:     "skips pre-release versions",
			tags:     []string{"1.0.0", "2.0.0-rc1", "1.5.0", "2.0.0-beta.1"},
			expected: "1.5.0",
		},
		{
			name:    "no valid semver tags",
			tags:    []string{"latest", "main", "sha-abc123"},
			wantErr: true,
		},
		{
			name:    "empty tag list",
			tags:    []string{},
			wantErr: true,
		},
		{
			name:     "single tag",
			tags:     []string{"3.0.0"},
			expected: "3.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LatestSemverTag(tc.tags)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}
