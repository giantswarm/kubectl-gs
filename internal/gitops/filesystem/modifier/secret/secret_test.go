package secret

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_SecretModifier(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier SecretModifier
	}{
		{
			name: "add key to the secret",
			expected: []byte(`apiVersion: v1
data:
  master.123456789ABCDEF.asc: RkFLRSBQVUJMSUMgS0VZIE1BVEVSSUFM
kind: Secret
metadata:
  name: sops-gpg-master
  namespace: default
`),
			input: []byte(`apiVersion: v1
kind: Secret
metadata:
  name: sops-gpg-master
  namespace: default
`),
			modifier: SecretModifier{
				KeysToAdd: map[string]string{
					"master.123456789ABCDEF.asc": "FAKE PUBLIC KEY MATERIAL",
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
