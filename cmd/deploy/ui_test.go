package deploy

import (
	"strings"
	"testing"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListVersionsOutput_SortsByVersionThenCatalog(t *testing.T) {
	testCases := []struct {
		name          string
		entries       []applicationv1alpha1.AppCatalogEntry
		expectedOrder []string // Expected version and catalog in order: "version:catalog"
		deployedVer   string
		deployedCat   string
	}{
		{
			name: "sorts by catalog ascending, then version descending",
			entries: []applicationv1alpha1.AppCatalogEntry{
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "1.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "2.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "community",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "2.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "1.5.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "2.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "azure",
						},
					},
				},
			},
			expectedOrder: []string{
				"2.0.0:azure",
				"2.0.0:community",
				"2.0.0:giantswarm",
				"1.5.0:giantswarm",
				"1.0.0:giantswarm",
			},
		},
		{
			name: "sorts with same version across multiple catalogs",
			entries: []applicationv1alpha1.AppCatalogEntry{
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "3.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "zzz-catalog",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "3.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "aaa-catalog",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "3.0.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "mmm-catalog",
						},
					},
				},
			},
			expectedOrder: []string{
				"3.0.0:aaa-catalog",
				"3.0.0:mmm-catalog",
				"3.0.0:zzz-catalog",
			},
		},
		{
			name: "real-world example with control-plane-catalog and giantswarm",
			entries: []applicationv1alpha1.AppCatalogEntry{
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.14.1",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.14.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "control-plane-catalog",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.14.1",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "control-plane-catalog",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.13.4",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.13.4",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "control-plane-catalog",
						},
					},
				},
				{
					Spec: applicationv1alpha1.AppCatalogEntrySpec{
						Version: "0.14.0",
						Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
							Name: "giantswarm",
						},
					},
				},
			},
			expectedOrder: []string{
				"0.14.1:control-plane-catalog",
				"0.14.0:control-plane-catalog",
				"0.13.4:control-plane-catalog",
				"0.14.1:giantswarm",
				"0.14.0:giantswarm",
				"0.13.4:giantswarm",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entries := &applicationv1alpha1.AppCatalogEntryList{
				Items: tc.entries,
			}

			output := ListVersionsOutput("test-app", entries, tc.deployedVer, tc.deployedCat)

			// Parse the output to extract the order of versions and catalogs
			lines := strings.Split(output, "\n")
			var actualOrder []string

			for _, line := range lines {
				// Look for lines that contain version info
				// Format is: "  • VERSION [CATALOG]"
				if strings.Contains(line, "•") && strings.Contains(line, "[") {
					// Extract version and catalog
					// Remove ANSI color codes for testing
					cleanLine := removeANSICodes(line)
					// Parse the line to extract version and catalog
					parts := strings.Fields(cleanLine)
					if len(parts) >= 3 {
						version := parts[1]
						// Extract catalog from [catalog] format
						catalogPart := parts[len(parts)-1]
						catalog := strings.Trim(catalogPart, "[]")
						actualOrder = append(actualOrder, version+":"+catalog)
					}
				}
			}

			// Verify the order matches expectations
			if len(actualOrder) != len(tc.expectedOrder) {
				t.Fatalf("Expected %d entries, got %d.\nExpected: %v\nActual: %v",
					len(tc.expectedOrder), len(actualOrder), tc.expectedOrder, actualOrder)
			}

			for i, expected := range tc.expectedOrder {
				if actualOrder[i] != expected {
					t.Errorf("At position %d: expected %s, got %s", i, expected, actualOrder[i])
				}
			}
		})
	}
}

// removeANSICodes removes ANSI color codes from a string for easier testing
func removeANSICodes(s string) string {
	// Simple approach: iterate through and remove escape sequences
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

func TestListVersionsOutput_EmptyList(t *testing.T) {
	entries := &applicationv1alpha1.AppCatalogEntryList{
		Items: []applicationv1alpha1.AppCatalogEntry{},
	}

	output := ListVersionsOutput("test-app", entries, "", "")

	if !strings.Contains(output, "No versions found") {
		t.Errorf("Expected 'No versions found' message, got: %s", output)
	}
}

func TestListVersionsOutput_MarksDeployedVersion(t *testing.T) {
	entries := &applicationv1alpha1.AppCatalogEntryList{
		Items: []applicationv1alpha1.AppCatalogEntry{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app-1.0.0",
				},
				Spec: applicationv1alpha1.AppCatalogEntrySpec{
					Version: "1.0.0",
					Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
						Name: "giantswarm",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app-2.0.0",
				},
				Spec: applicationv1alpha1.AppCatalogEntrySpec{
					Version: "2.0.0",
					Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
						Name: "giantswarm",
					},
				},
			},
		},
	}

	output := ListVersionsOutput("test-app", entries, "1.0.0", "giantswarm")

	// Should contain deployed marker for version 1.0.0
	if !strings.Contains(output, "deployed") {
		t.Errorf("Expected output to mark deployed version, got: %s", output)
	}
}

func TestStatusOutput_WithVersionInfo(t *testing.T) {
	testCases := []struct {
		name                    string
		kustomizationsReady     bool
		notReadyKustomizations  []resourceInfo
		suspendedKustomizations []resourceInfo
		suspendedApps           []resourceInfo
		suspendedGitRepos       []resourceInfo
		expectedContains        []string
	}{
		{
			name:                    "all healthy",
			kustomizationsReady:     true,
			notReadyKustomizations:  []resourceInfo{},
			suspendedKustomizations: []resourceInfo{},
			suspendedApps:           []resourceInfo{},
			suspendedGitRepos:       []resourceInfo{},
			expectedContains:        []string{"All Systems Healthy"},
		},
		{
			name:                    "suspended apps with version info",
			kustomizationsReady:     true,
			notReadyKustomizations:  []resourceInfo{},
			suspendedKustomizations: []resourceInfo{},
			suspendedApps: []resourceInfo{
				{
					name:      "my-app",
					namespace: "default",
					version:   "1.2.3",
					catalog:   "giantswarm",
					status:    "deployed",
				},
				{
					name:      "another-app",
					namespace: "org-example",
					version:   "2.0.0",
					catalog:   "control-plane-catalog",
					status:    "deployed",
				},
			},
			suspendedGitRepos: []resourceInfo{},
			expectedContains: []string{
				"Suspended Apps",
				"my-app",
				"1.2.3",
				"giantswarm",
				"deployed",
				"another-app",
				"2.0.0",
				"control-plane-catalog",
				"NAME",
				"NAMESPACE",
				"VERSION",
				"CATALOG",
				"STATUS",
			},
		},
		{
			name:                    "suspended git repos with branch info",
			kustomizationsReady:     true,
			notReadyKustomizations:  []resourceInfo{},
			suspendedKustomizations: []resourceInfo{},
			suspendedApps:           []resourceInfo{},
			suspendedGitRepos: []resourceInfo{
				{
					name:      "config-repo",
					namespace: "default",
					branch:    "main",
					url:       "https://github.com/giantswarm/config-repo",
					status:    "Ready",
				},
				{
					name:      "another-config",
					namespace: "org-example",
					branch:    "v1.0.0",
					url:       "https://github.com/giantswarm/another-config",
					status:    "Ready",
				},
			},
			expectedContains: []string{
				"Suspended Git Repositories",
				"config-repo",
				"main",
				"https://github.com/giantswarm/config-repo",
				"Ready",
				"another-config",
				"v1.0.0",
				"https://github.com/giantswarm/another-config",
				"NAME",
				"NAMESPACE",
				"BRANCH",
				"URL",
				"STATUS",
			},
		},
		{
			name:                    "apps without version info",
			kustomizationsReady:     true,
			notReadyKustomizations:  []resourceInfo{},
			suspendedKustomizations: []resourceInfo{},
			suspendedApps: []resourceInfo{
				{
					name:      "app-no-version",
					namespace: "default",
					version:   "",
					catalog:   "",
					status:    "Unknown",
				},
			},
			suspendedGitRepos: []resourceInfo{},
			expectedContains: []string{
				"Suspended Apps",
				"app-no-version",
				"-", // Should show dash for empty values
			},
		},
		{
			name:                   "suspended kustomizations",
			kustomizationsReady:    true,
			notReadyKustomizations: []resourceInfo{},
			suspendedKustomizations: []resourceInfo{
				{
					name:      "flux-system",
					namespace: "flux-system",
				},
				{
					name:      "my-app-kustomization",
					namespace: "default",
				},
			},
			suspendedApps:     []resourceInfo{},
			suspendedGitRepos: []resourceInfo{},
			expectedContains: []string{
				"Suspended Kustomizations",
				"flux-system",
				"my-app-kustomization",
				"NAME",
				"NAMESPACE",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := StatusOutput(
				tc.kustomizationsReady,
				tc.notReadyKustomizations,
				tc.suspendedKustomizations,
				tc.suspendedApps,
				tc.suspendedGitRepos,
			)

			// Remove ANSI codes for easier testing
			cleanOutput := removeANSICodes(output)

			for _, expected := range tc.expectedContains {
				if !strings.Contains(cleanOutput, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, cleanOutput)
				}
			}
		})
	}
}
