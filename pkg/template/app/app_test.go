package app

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

func Test_NewAppCR(t *testing.T) {
	testCases := []struct {
		name               string
		config             Config
		expectedGoldenFile string
	}{
		{
			name: "case 0: flawless flow",
			config: Config{
				AppName:           "nginx-ingress-controller-app",
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
				AppName:           "nginx-ingress-controller-app",
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
				AppName:                 "nginx-ingress-controller-app",
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
				AppName:              "nginx-ingress-controller-app",
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
		{
			name: "case 4: override app name",
			config: Config{
				AppName:           "internal-nginx-ingress-controller",
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: true,
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_override_app_name_yaml_output.golden",
		},
		{
			name: "case 5: flawless flow for organization",
			config: Config{
				AppName:           "nginx-ingress-controller-app",
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: true,
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				Organization:      "giantswarm",
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_flawless_flow_organization_yaml_output.golden",
		},
		{
			name: "case 6: defaulting disabled for organization",
			config: Config{
				AppName:           "nginx-ingress-controller-app",
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: false,
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				Organization:      "giantswarm",
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_defaulting_disabled_organization_yaml_output.golden",
		},
		{
			name: "case 7: flawless with timeouts",
			config: Config{
				AppName:           "nginx-ingress-controller-app",
				Catalog:           "giantswarm",
				Cluster:           "eggs2",
				DefaultingEnabled: true,
				InstallTimeout:    &metav1.Duration{Duration: 6 * time.Minute},
				Name:              "nginx-ingress-controller-app",
				Namespace:         "kube-system",
				RollbackTimeout:   &metav1.Duration{Duration: 7 * time.Minute},
				UninstallTimeout:  &metav1.Duration{Duration: 8 * time.Minute},
				UpgradeTimeout:    &metav1.Duration{Duration: 9 * time.Minute},
				Version:           "1.17.0",
			},
			expectedGoldenFile: "app_flawless_flow_timeouts_yaml_output.golden",
		},
		{
			name: "case 8: app config points to cluster-values configmap",
			config: Config{
				AppName:                "eggs2-default-apps",
				Catalog:                "cluster",
				Cluster:                "eggs2",
				DefaultingEnabled:      false,
				InCluster:              true,
				Name:                   "default-apps-gcp",
				Namespace:              "org-giantswarm",
				Version:                "0.13.0",
				UseClusterValuesConfig: true,
			},
			expectedGoldenFile: "app_config_cluster_values_yaml_output.golden",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			gf := goldenfile.New("testdata", tc.expectedGoldenFile)
			expectedResult, err := gf.Read()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			appCRYaml, err := NewAppCR(tc.config)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if !cmp.Equal(appCRYaml, expectedResult) {
				t.Fatalf("\n\n%s\n", cmp.Diff(string(expectedResult), string(appCRYaml)))
			}
		})
	}
}
