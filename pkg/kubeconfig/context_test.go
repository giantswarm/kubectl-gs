package kubeconfig

import (
	"strconv"
	"testing"
)

func TestGenerateKubeContextName(t *testing.T) {
	codeName := "test"
	result := GenerateKubeContextName(codeName)
	expected := "gs-test"

	if result != expected {
		t.Fatalf("Value not expected, got: %s", result)
	}
}

func TestIsKubeContext(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "case 0: check kube context, with correct prefix",
			input:    "gs-test",
			expected: true,
		},
		{
			name:     "case 1: check kube context, with no prefix",
			input:    "test",
			expected: false,
		},
		{
			name:     "case 2: check kube context, with incorrect prefix",
			input:    "ms-test",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := IsKubeContext(tc.input)

			if result != tc.expected {
				t.Fatalf("Value not expected, got: %t", result)
			}
		})
	}
}

func TestGetCodeNameFromKubeContext(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "case 0: get installation code name, with correct prefix",
			input:    "gs-test",
			expected: "test",
		},
		{
			name:     "case 1: get installation code name, with workload cluster context",
			input:    "gs-test-somecluster",
			expected: "test",
		},
		{
			name:     "case 2: get installation code name, with no prefix",
			input:    "test",
			expected: "test",
		},
		{
			name:     "case 3: get installation code name, with incorrect prefix",
			input:    "ms-test",
			expected: "ms-test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetCodeNameFromKubeContext(tc.input)

			if result != tc.expected {
				t.Fatalf("Value not expected, got: %s", result)
			}
		})
	}
}

func TestIsCodeName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "case 0: check if it's an installation codename, with no special characters",
			input:    "test",
			expected: true,
		},
		{
			name:     "case 1: check if it's an installation codename, with URL form",
			input:    "test.com",
			expected: false,
		},
		{
			name:     "case 2: check if it's an installation codename, with other URL form",
			input:    "https://test",
			expected: false,
		},
		{
			name:     "case 3: check if it's an installation codename, with context name",
			input:    "gs-test",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCodeName(tc.input)

			if result != tc.expected {
				t.Fatalf("Value not expected, got: %t", result)
			}
		})
	}
}

func TestGetKubeContextType(t *testing.T) {
	testCases := []struct {
		input    string
		expected ContextType
	}{
		{
			input:    "gs-test",
			expected: ContextTypeMC,
		},
		{
			input:    "test123",
			expected: ContextTypeNone,
		},
		{
			input:    "gs-some-wc",
			expected: ContextTypeWC,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := GetKubeContextType(tc.input)

			if result != tc.expected {
				t.Fatalf("Expected %d, got: %d", tc.expected, result)
			}
		})
	}
}

func TestGetClusterNameFromKubeContext(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    "gs-test",
			expected: "",
		},
		{
			input:    "gs-test-wc",
			expected: "wc",
		},
		{
			input:    "gs-some-wc-213-sad",
			expected: "wc-213-sad",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := GetClusterNameFromKubeContext(tc.input)

			if result != tc.expected {
				t.Fatalf("Expected %s, got: %s", tc.expected, result)
			}
		})
	}
}
