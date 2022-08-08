package app

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_AppModifier(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier AppModifier
	}{
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
				ImagePolicyToAdd: "demowc-hello-world",
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
				ImagePolicyToAdd: "demowc-hello-world",
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
