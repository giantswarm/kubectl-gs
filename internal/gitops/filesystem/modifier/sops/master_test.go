package sigskus

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	masterRegex = `management-clusters/demomc/secrets/.*\.enc\.yaml`
	keysRegex   = `management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml`
)

func Test_MasterKeyRuleModifier(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "ahead of an organisation key's rule for the same directory",
			input: `creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: MASTER
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/organizations/demoorg/namespace/secrets/.*\.enc\.yaml
  pgp: ORG
`,
			expected: `creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml
  pgp: MASTER
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: MASTER
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/organizations/demoorg/namespace/secrets/.*\.enc\.yaml
  pgp: ORG
`,
		},
		{
			name: "already there",
			input: `creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: MASTER
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml
  pgp: MASTER
`,
			expected: `creation_rules:
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/secrets/.*\.enc\.yaml
  pgp: MASTER
- encrypted_regex: ^(data|stringData)$
  path_regex: management-clusters/demomc/organizations/[^/]+/namespace/secrets/.*\.(gpg|age)key\.enc\.yaml
  pgp: MASTER
`,
		},
		{
			name:        "without a master key",
			input:       "creation_rules: []\n",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := MasterKeyRuleModifier{MasterPathRegex: masterRegex, PathRegex: keysRegex}

			out, err := m.Execute([]byte(tc.input))
			if tc.expectError {
				if !IsMasterKeyNotFound(err) {
					t.Fatalf("expected a missing master key error, got: %v", err)
				}
				return
			}

			checkMasterKeyRule(t, tc.input, tc.expected, out, err)
		})
	}
}

func checkMasterKeyRule(t *testing.T, input, expected string, out []byte, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, string(out)); diff != "" {
		t.Errorf(".sops.yaml (-want +got):\n%s", diff)
	}

	ok, err := HasMasterKey([]byte(input), masterRegex)
	if err != nil || !ok {
		t.Errorf("expected the master key to be found, got %t, %v", ok, err)
	}
}
