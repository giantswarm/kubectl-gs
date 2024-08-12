package commonconfig

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v4/internal/key"
)

func TestCommonConfig_GetProviderFromInstallation(t *testing.T) {
	testCases := []struct {
		name             string
		k8sApiURL        string
		resolvedProvider string
		expectedResult   string
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
			name:           "case 2: AWS URL",
			k8sApiURL:      "https://g8s.test.eu-west-1.aws.coolio.com",
			expectedResult: key.ProviderAWS,
		},
		{
			name:           "case 3: Azure URL",
			k8sApiURL:      "https://g8s.test.eu-west-1.azure.coolio.com",
			expectedResult: key.ProviderAzure,
		},
		{
			name:           "case 4: Teleport URL",
			k8sApiURL:      "https://teleport.coolio.io",
			expectedResult: key.ProviderDefault,
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
			result, err := cc.GetProviderFromConfig(ctx, mockAthenaServer.URL)

			if err != nil {
				t.Fatal(err)
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
