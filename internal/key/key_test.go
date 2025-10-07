package key

import (
	"fmt"
	"testing"
)

func Test_ValidateName(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedResult bool
	}{
		{
			name:           "Empty string",
			input:          "",
			expectedResult: false,
		},
		{
			name:           "1 character",
			input:          "a",
			expectedResult: true,
		},
		{
			name:           "1 number",
			input:          "7",
			expectedResult: false,
		},
		{
			name:           "Cant start with dash",
			input:          "-golem",
			expectedResult: false,
		},
		{
			name:           "Cant end with dash",
			input:          "golem12-",
			expectedResult: false,
		},
		{
			name:           "Simple name",
			input:          "golem",
			expectedResult: true,
		},
		{
			name:           "Simple name, max length",
			input:          "aaaaabbbbbaaaaabbbbb",
			expectedResult: true,
		},
		{
			name:           "Simple name, too long",
			input:          "aaaaabbbbbaaaaabbbbbx",
			expectedResult: false,
		},
		{
			name:           "Complex name",
			input:          "example12",
			expectedResult: true,
		},
		{
			name:           "Name with dashes",
			input:          "example-staging-42",
			expectedResult: true,
		},
		{
			name:           "Name with dashes, max length",
			input:          "example-dev-hello-42",
			expectedResult: true,
		},
		{
			name:           "Name with dashes, too long",
			input:          "example-dev-hello-world",
			expectedResult: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			result, err := ValidateName(tc.input)

			if result != tc.expectedResult {
				t.Fatalf("expected result: %v, got: %v, error: %v", tc.expectedResult, result, err)
			}
		})
	}
}
