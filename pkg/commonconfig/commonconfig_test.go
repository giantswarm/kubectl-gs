package commonconfig

import (
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

func TestCommonConfig_GetProvider(t *testing.T) {
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
			result, err := cc.GetProvider()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if result != tc.expectedResult {
				t.Fatalf("value not expected, got: %s", result)
			}
		})
	}
}
