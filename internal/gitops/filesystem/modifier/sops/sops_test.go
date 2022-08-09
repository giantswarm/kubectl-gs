package sigskus

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_SopsModifier(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier SopsModifier
	}{
		{
			name: "add rule",
			expected: []byte(`creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: 123456789ABCDEF
`),
			input: []byte(`creation_rules: []`),
			modifier: SopsModifier{
				RulesToAdd: []map[string]interface{}{
					map[string]interface{}{
						"encrypted_regex": "^(data|stringData)$",
						"path_regex":      "management-clusters/demomc/secrets/.*\\.enc\\.yaml",
						"pgp":             "123456789ABCDEF",
					},
				},
			},
		},
		{
			name: "do not add already existing rule",
			expected: []byte(`creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: 123456789ABCDEF
`),
			input: []byte(`creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: 123456789ABCDEF
`),
			modifier: SopsModifier{
				RulesToAdd: []map[string]interface{}{
					map[string]interface{}{
						"encrypted_regex": "^(data|stringData)$",
						"path_regex":      "management-clusters/demomc/secrets/.*\\.enc\\.yaml",
						"pgp":             "123456789ABCDEF",
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			got, err := tc.modifier.Execute(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if !bytes.Equal(got, tc.expected) {
				t.Fatalf("want matching files \n%s\n", cmp.Diff(string(tc.expected), string(got)))
			}
		})
	}
}
