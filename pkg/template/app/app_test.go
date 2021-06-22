package app

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

func Test_NewAppCR(t *testing.T) {
	testCases := []struct {
		name               string
		config             Config
		expectedGoldenFile string
		errorMatcher       func(err error) bool
	}{
		{
			name: "case 0: flawless flow",
			config: Config{
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: true,
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_flawless_flow_yaml_output.golden",
		},
		{
			name: "case 1: defaulting disabled",
			config: Config{
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: false,
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_defaulting_disabled_yaml_output.golden",
		},
		{
			name: "case 2: user values",
			config: Config{
				Catalog:                 "giantswarm",
				Cluster:                 "eggs2",
				DefaultingEnabled:       true,
				Name:                    "nginx-ingress-controller-app",
				Namespace:               "kube-system",
				UserConfigConfigMapName: "nginx-ingress-controller-app-user-values",
				Version:                 "1.17.0",
			},
			expectedGoldenFile: "app_user_values_yaml_output.golden",
		},
		{
			name: "case 3: user secrets with defauting disabled",
			config: Config{
				Catalog:              "giantswarm",
				Cluster:              "eggs2",
				DefaultingEnabled:    false,
				Name:                 "nginx-ingress-controller-app",
				Namespace:            "kube-system",
				UserConfigSecretName: "nginx-ingress-controller-app-user-secrets",
				Version:              "1.17.0",
			},
			expectedGoldenFile: "app_user_secrets_yaml_output.golden",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			appCRYaml, err := NewAppCR(tc.config)

			gf := goldenfile.New("testdata", tc.expectedGoldenFile)
			expectedResult, err := gf.Read()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !cmp.Equal(appCRYaml, expectedResult) {
				t.Fatalf("\n\n%s\n", cmp.Diff(string(expectedResult), string(appCRYaml)))
			}
		})
	}
}
