package kustomization

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_KustomizationModifier(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier KustomizationModifier
	}{
		{
			name: "add resource",
			expected: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- demowc.yaml
`),
			input: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []`),
			modifier: KustomizationModifier{
				ResourcesToAdd: []string{
					"demowc.yaml",
				},
			},
		},
		{
			name: "do not add if present",
			expected: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- demowc.yaml
`),
			input: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- demowc.yaml
`),
			modifier: KustomizationModifier{
				ResourcesToAdd: []string{
					"demowc.yaml",
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
