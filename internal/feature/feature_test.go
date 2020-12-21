package feature

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestService_Supports(t *testing.T) {
	testCases := []struct {
		name           string
		provider       string
		feature        string
		version        string
		expectedResult bool
	}{
		{
			name:           "case 0: autoscaling on aws 10.0.0",
			provider:       ProviderAWS,
			feature:        Autoscaling,
			version:        "10.0.0",
			expectedResult: true,
		},
		{
			name:           "case 1: autoscaling on aws 11.0.0",
			provider:       ProviderAWS,
			feature:        Autoscaling,
			version:        "13.1.0",
			expectedResult: true,
		},
		{
			name:           "case 2: autoscaling on azure 13.0.0",
			provider:       ProviderAzure,
			feature:        Autoscaling,
			version:        "13.0.0",
			expectedResult: false,
		},
		{
			name:           "case 3: autoscaling on azure 13.1.0",
			provider:       ProviderAzure,
			feature:        Autoscaling,
			version:        "13.1.0",
			expectedResult: true,
		},
		{
			name:           "case 4: autoscaling on azure 13.2.0",
			provider:       ProviderAzure,
			feature:        Autoscaling,
			version:        "13.2.0",
			expectedResult: true,
		},
		{
			name:           "case 5: autoscaling on kvm 13.0.0",
			provider:       ProviderKVM,
			feature:        Autoscaling,
			version:        "13.0.0",
			expectedResult: false,
		},
		{
			name:           "case 6: nodepool conditions on aws 10.0.0",
			provider:       ProviderAWS,
			feature:        NodePoolConditions,
			version:        "10.0.0",
			expectedResult: false,
		},
		{
			name:           "case 7: nodepool conditions on azure 13.0.0",
			provider:       ProviderAzure,
			feature:        NodePoolConditions,
			version:        "13.0.0",
			expectedResult: true,
		},
		{
			name:           "case 8: nodepool conditions on kvm 13.0.0",
			provider:       ProviderKVM,
			feature:        NodePoolConditions,
			version:        "13.0.0",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capabilities := New(tc.provider)
			isSupported := capabilities.Supports(tc.feature, tc.version)

			diff := cmp.Diff(tc.expectedResult, isSupported)
			if len(diff) > 0 {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
}
