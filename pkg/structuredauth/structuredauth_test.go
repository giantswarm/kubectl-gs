package structuredauth

import (
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// testAuthConfig is the AuthenticationConfiguration the cluster chart renders
// with structured authentication: the legacy Dex issuer first, then the
// structuredAuthentication.issuers.
const testAuthConfig = `apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
jwt:
  - issuer:
      url: "https://dex.my-mc.example.com"
      audiences:
        - "dex-k8s-authenticator"
    claimMappings:

      username:
        claim: "email"
        prefix: ""
      groups:
        claim: "groups"
        prefix: ""
  - issuer:
      url: "https://token.actions.githubusercontent.com"
      audiences:
        - "my-mc"
        - "other"
      audienceMatchPolicy: "MatchAny"
      discoveryURL: "https://discovery.example.com"
      certificateAuthority: |
        -----BEGIN CERTIFICATE-----
        test
        -----END CERTIFICATE-----
    claimValidationRules:
      - 
        
        expression: "claims.repository == 'acme/repo'"
        message: "not acme/repo"
      - claim: "aud"
        requiredValue: "my-mc"
    claimMappings:

      username:
        expression: "'github.com/' + claims.actor"
      groups:
        claim: "groups"
        prefix: "gh:"
      uid:
        claim: "sub"
    userValidationRules:
      - expression: "!user.username.startsWith('system:')"
  - issuer:
      url: ""
      audiences:
        - "skipped"
  - issuer:
      url: "https://no-audience.example.com"
  - issuer:
      url: "https://future.example.com"
      audiences:
        - "future"
      egressSelectorType: "controlplane"
      requiredClaim: "x"
    claimMappings:
      username:
        claim: "email"
        caseFold: true
      roles:
        claim: "roles"
    futureRules:
      - expression: "true"
`

var testIssuers = []Issuer{
	{
		URL:       "https://dex.my-mc.example.com",
		Audiences: []string{"dex-k8s-authenticator"},
		Username:  ClaimMapping{Claim: "email"},
		Groups:    ClaimMapping{Claim: "groups"},
	},
	{
		URL:                  "https://token.actions.githubusercontent.com",
		Audiences:            []string{"my-mc", "other"},
		CertificateAuthority: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n",
		DiscoveryURL:         "https://discovery.example.com",
		Username:             ClaimMapping{Expression: "'github.com/' + claims.actor"},
		Groups:               ClaimMapping{Claim: "groups", Prefix: "gh:"},
		MapsUIDOrExtra:       true,
		ClaimValidationRules: 2,
		UserValidationRules:  1,
	},
	{URL: "https://no-audience.example.com"},
	{
		URL:                "https://future.example.com",
		Audiences:          []string{"future"},
		EgressSelectorType: "controlplane",
		Username:           ClaimMapping{Claim: "email"},
		UnknownFields:      []string{"futureRules", "claimMappings.roles", "claimMappings.username.caseFold", "issuer.requiredClaim"},
	},
}

func file(path, encoding, content string) interface{} {
	f := map[string]interface{}{"path": path, "content": content}
	if encoding != "" {
		f["encoding"] = encoding
	}
	return f
}

func kcpWithFiles(files ...interface{}) *unstructured.Unstructured {
	spec := map[string]interface{}{}
	if files != nil {
		spec["files"] = files
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{"kubeadmConfigSpec": spec},
	}}
}

func Test_Issuers(t *testing.T) {
	testCases := []struct {
		name            string
		kcp             *unstructured.Unstructured
		expectedIssuers []Issuer
		expectedError   bool
	}{
		{
			name: "case 0: base64-encoded file among others",
			kcp: kcpWithFiles(
				file("/etc/kubernetes/policies/audit-policy.yaml", "base64", base64.StdEncoding.EncodeToString([]byte("kind: Policy"))),
				file(ConfigPath, "base64", base64.StdEncoding.EncodeToString([]byte(testAuthConfig))),
			),
			expectedIssuers: testIssuers,
		},
		{
			name:            "case 1: plain file",
			kcp:             kcpWithFiles(file(ConfigPath, "", testAuthConfig)),
			expectedIssuers: testIssuers,
		},
		{
			name: "case 2: no files",
			kcp:  kcpWithFiles(),
		},
		{
			name: "case 3: no authentication configuration file",
			kcp:  kcpWithFiles(file("/etc/kubernetes/policies/audit-policy.yaml", "", "kind: Policy")),
		},
		{
			name: "case 4: no JWT issuer",
			kcp:  kcpWithFiles(file(ConfigPath, "", "apiVersion: apiserver.config.k8s.io/v1\nkind: AuthenticationConfiguration\n")),
		},
		{
			name:          "case 5: invalid base64",
			kcp:           kcpWithFiles(file(ConfigPath, "base64", "not base64!")),
			expectedError: true,
		},
		{
			name:          "case 6: unsupported encoding",
			kcp:           kcpWithFiles(file(ConfigPath, "gzip+base64", "H4sI")),
			expectedError: true,
		},
		{
			name:          "case 7: invalid YAML",
			kcp:           kcpWithFiles(file(ConfigPath, "", "jwt: [")),
			expectedError: true,
		},
		{
			name:          "case 8: JWT authenticator that is not an object",
			kcp:           kcpWithFiles(file(ConfigPath, "", "jwt:\n  - https://dex.my-mc.example.com\n")),
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issuers, err := Issuers(tc.kcp)
			if tc.expectedError {
				if err == nil {
					t.Fatalf("expected an error, got issuers %v", issuers)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(tc.expectedIssuers, issuers); diff != "" {
				t.Fatalf("issuers differ (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_IssuersAt(t *testing.T) {
	kcp := kcpWithFiles(file("/etc/other/auth.yaml", "", testAuthConfig))

	issuers, err := IssuersAt(kcp, "/etc/other/auth.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(testIssuers, issuers); diff != "" {
		t.Fatalf("issuers differ (-want +got):\n%s", diff)
	}

	issuers, err = Issuers(kcp)
	if err != nil || issuers != nil {
		t.Fatalf("expected no issuer at ConfigPath, got %v, %v", issuers, err)
	}
}
