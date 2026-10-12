package flags

import (
	"strings"
	"testing"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
)

const testIssuerURL = "https://idp.example.com"

func Test_validateOIDC(t *testing.T) {
	testCases := []struct {
		name          string
		disableOIDC   bool
		oidc          common.OIDC
		expectedError string
	}{
		{
			name: "case 0: default",
		},
		{
			name:        "case 1: disabled",
			disableOIDC: true,
		},
		{
			name: "case 2: explicit issuer and client id",
			oidc: common.OIDC{IssuerURL: testIssuerURL, ClientID: "my-client"},
		},
		{
			name: "case 3: explicit claim only",
			oidc: common.OIDC{GroupsClaim: "roles"},
		},
		{
			name:          "case 4: explicit issuer without client id",
			oidc:          common.OIDC{IssuerURL: testIssuerURL},
			expectedError: "--oidc-client-id must not be empty when specifying --oidc-issuer-url",
		},
		{
			name:          "case 5: disabled with an explicit issuer",
			disableOIDC:   true,
			oidc:          common.OIDC{IssuerURL: testIssuerURL, ClientID: "my-client"},
			expectedError: "--disable-oidc and --oidc-issuer-url are mutually exclusive",
		},
		{
			name:          "case 6: disabled with an explicit claim",
			disableOIDC:   true,
			oidc:          common.OIDC{UsernameClaim: "sub"},
			expectedError: "--disable-oidc and --oidc-username-claim are mutually exclusive",
		},
		{
			name:          "case 7: disabled with a CA file",
			disableOIDC:   true,
			oidc:          common.OIDC{CAFile: "ca.pem"},
			expectedError: "--disable-oidc and --oidc-ca-file are mutually exclusive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &Flag{DisableOIDC: tc.disableOIDC, OIDC: tc.oidc}

			err := f.validateOIDC()
			if tc.expectedError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.expectedError) || microerror.Cause(err) != invalidFlagError {
				t.Fatalf("expected invalid flag error containing %q, got %v", tc.expectedError, err)
			}
		})
	}
}
