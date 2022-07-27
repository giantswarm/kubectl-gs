package creator

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_Execute(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier Modifier
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
		{
			name: "add comment for automatic updates",
			expected: []byte(`apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: demowc-hello-world
spec:
  version: 0.3.0 # {"$imagepolicy": "default:demowc-hello-world:tag"}
  catalog: giantswarm
  name: hello-world
  namespace: default
`),
			input: []byte(`apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: demowc-hello-world
spec:
  version: 0.3.0
  catalog: giantswarm
  name: hello-world
  namespace: default
`),
			modifier: AppModifier{
				ImagePolicy: "demowc-hello-world",
			},
		},
		{
			name: "do not change already existing automatic updates",
			expected: []byte(`apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: demowc-hello-world
spec:
  version: 0.3.0 # {"$imagepolicy": "default:demowc-hello-world:tag"}
  catalog: giantswarm
  name: hello-world
  namespace: default
`),
			input: []byte(`apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: demowc-hello-world
spec:
  version: 0.3.0 # {"$imagepolicy": "default:demowc-hello-world:tag"}
  catalog: giantswarm
  name: hello-world
  namespace: default
`),
			modifier: AppModifier{
				ImagePolicy: "demowc-hello-world",
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
