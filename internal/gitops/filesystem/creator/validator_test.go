package creator

import (
	"fmt"
	"strings"
	"testing"
)

func Test_Validators(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError string
		input         []byte
		validator     Validator
	}{
		{
			name:          "do not fail on empty resources",
			expectedError: "",
			input: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []`),
			validator: KustomizationValidator{
				ReferencesBase: true,
			},
		},
		{
			name:          "do not fail when base is not referenced",
			expectedError: "",
			input: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- appcr.yaml
- configmap.yaml
`),
			validator: KustomizationValidator{
				ReferencesBase: true,
			},
		},
		{
			name:          "fail when base is referenced",
			expectedError: "validation error: Operation cannot be fulfilled on the object coming from a base. Operation should be conducted on the base instead.",
			input: []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../../../../../../bases/app/hello-world
`),
			validator: KustomizationValidator{
				ReferencesBase: true,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			err := tc.validator.Execute(tc.input)

			switch {
			case err != nil && tc.expectedError == "":
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.expectedError != "":
				t.Fatalf("error == nil, want non-nil")
			}

			if err != nil && tc.expectedError != "" {
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("error == %#v, want %#v ", err.Error(), tc.expectedError)
				}

			}
		})
	}
}
