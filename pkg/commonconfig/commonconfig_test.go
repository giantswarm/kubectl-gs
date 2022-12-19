package commonconfig

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

func TestCommonConfig_GetProviderFromConfig(t *testing.T) {
	testCases := []struct {
		name           string
		k8sApiURL      string
		expectedResult string
	}{
		{
			name:           "case 0: AWS url",
			k8sApiURL:      "https://g8s.test.eu-west-1.aws.coolio.com",
			expectedResult: key.ProviderAWS,
		},
		{
			name:           "case 1: Azure url",
			k8sApiURL:      "https://g8s.test.eu-west-1.azure.coolio.com",
			expectedResult: key.ProviderAzure,
		},
		{
			name:           "case 2: OpenStack url",
			k8sApiURL:      "https://test12.customer.coolio.com",
			expectedResult: key.ProviderOpenStack,
		},
		{
			name:           "case 3: URL containing 'aws', but not AWS",
			k8sApiURL:      "https://aws12.customer.coolio.com",
			expectedResult: key.ProviderOpenStack,
		},
		{
			name:           "case 4: URL containing 'azure', but not Azure",
			k8sApiURL:      "https://azure12.customer.coolio.com",
			expectedResult: key.ProviderOpenStack,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cflags := genericclioptions.NewConfigFlags(false)
			*cflags.APIServer = tc.k8sApiURL

			cc := New(cflags)
			result, err := cc.GetProviderFromConfig()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if result != tc.expectedResult {
				t.Fatalf("value not expected, got: %s", result)
			}
		})
	}
}

func TestCommonConfig_GetProviderFromInstallation(t *testing.T) {
	testCases := []struct {
		name             string
		k8sApiURL        string
		resolvedProvider string
		expectedResult   string
		expectedError    bool
		fallbackToConfig bool
	}{
		{
			name:             "case 0: AWS URL resolved as Capa",
			k8sApiURL:        "https://g8s.test.eu-west-1.aws.coolio.com",
			resolvedProvider: key.ProviderCAPA,
			expectedResult:   key.ProviderCAPA,
		},
		{
			name:             "case 1: Unknown URL resolved as Azure",
			k8sApiURL:        "https://api.azure12.customer.coolio.com",
			resolvedProvider: key.ProviderAzure,
			expectedResult:   key.ProviderAzure,
		},
		{
			name:          "case 2: Azure URL resolved with no fallback as error",
			k8sApiURL:     "https://g8s.test.eu-west-1.azure.coolio.com",
			expectedError: true,
		},
		{
			name:             "case 3: Azure URL resolved to error with fallback",
			k8sApiURL:        "https://g8s.test.eu-west-1.azure.coolio.com",
			fallbackToConfig: true,
			expectedResult:   key.ProviderAzure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hf := func(w http.ResponseWriter, r *http.Request) {
				body := graphqlResponseBody(tc.resolvedProvider)
				if len(body) == 0 {
					w.WriteHeader(404)
				} else {
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
				}
				_, _ = w.Write(graphqlResponseBody(tc.resolvedProvider))
			}
			mockAthenaServer := httptest.NewServer(http.HandlerFunc(hf))
			defer mockAthenaServer.Close()

			cflags := genericclioptions.NewConfigFlags(false)
			*cflags.APIServer = tc.k8sApiURL

			ctx := context.TODO()

			cc := New(cflags)
			result, err := cc.GetProviderFromInstallation(ctx, mockAthenaServer.URL, tc.fallbackToConfig)

			if err != nil && !tc.expectedError {
				t.Fatal(err)
			}
			if tc.expectedError && err == nil {
				t.Fatal("Expected error, received success")
			}
			if tc.expectedResult != result {
				t.Fatalf("Incorrect provider: expected %s, received %s", tc.expectedResult, result)
			}

		})
	}
}

func graphqlResponseBody(provider string) []byte {
	if provider == "" {
		return []byte{}
	}
	return []byte(fmt.Sprintf(`{"data":{"identity":{"provider":"%s","codename":"codename"}}}`, provider))
}
