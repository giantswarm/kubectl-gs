package common

import "testing"

func TestIsReleaseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "release version 34.0.0 returns true",
			version:  "34.0.0",
			expected: true,
		},
		{
			name:     "release version v34.0.0 with v prefix returns true",
			version:  "v34.0.0",
			expected: true,
		},
		{
			name:     "release version 35.1.2 returns true",
			version:  "35.1.2",
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
			name:     "chart version 33.9.9 returns false",
			version:  "33.9.9",
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
