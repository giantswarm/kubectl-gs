package ocirepo

import (
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
)

func Test_SemverModifier(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name: "replaces the tag",
			input: `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: demowc-hello-world
spec:
  # The chart kubectl-gs added.
  ref:
    tag: 0.3.0
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world
`,
			expected: `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: demowc-hello-world
spec:
  # The chart kubectl-gs added.
  ref:
    semver: '>=0.0.0'
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world
`,
		},
		{
			name: "adds a reference",
			input: `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
spec:
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world
`,
			expected: `apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
spec:
  url: oci://gsoci.azurecr.io/charts/giantswarm/hello-world
  ref:
    semver: '>=0.0.0'
`,
		},
		{
			name:      "not an OCIRepository",
			input:     "apiVersion: application.giantswarm.io/v1alpha1\nkind: App\nspec: {}\n",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SemverModifier{Range: ">=0.0.0"}.Execute([]byte(tc.input))
			if tc.expectErr {
				if microerror.Cause(err) != invalidConfigError {
					t.Fatalf("expected invalid config error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected output (-want +got):\n%s", diff)
			}
		})
	}
}
