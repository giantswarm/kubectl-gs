package common

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/giantswarm/kubectl-gs/v6/pkg/structuredauth"
)

// newKubeadmControlPlane returns a KubeadmControlPlane whose API server
// carries the given extra arguments.
const (
	testGroup     = "controlplane.cluster.x-k8s.io"
	testKind      = "KubeadmControlPlane"
	testIssuerURL = "https://dex.my-mc.example.com"
	testClientID  = "dex-k8s-authenticator"
)

// arg is one API server extra argument in the v1beta2 list form.
func arg(name, value string) interface{} {
	return map[string]interface{}{"name": name, "value": value}
}

func newKubeadmControlPlane(version, namespace, name string, extraArgs interface{}) *unstructured.Unstructured {
	kcp := &unstructured.Unstructured{Object: map[string]interface{}{
		"spec": map[string]interface{}{
			"kubeadmConfigSpec": map[string]interface{}{
				"clusterConfiguration": map[string]interface{}{
					"apiServer": map[string]interface{}{
						"extraArgs": extraArgs,
					},
				},
			},
		},
	}}
	kcp.SetGroupVersionKind(schema.GroupVersionKind{Group: testGroup, Version: version, Kind: testKind})
	kcp.SetNamespace(namespace)
	kcp.SetName(name)
	return kcp
}

func dexArgs() []interface{} {
	return []interface{}{
		arg("audit-log-maxage", "30"),
		arg("oidc-issuer-url", testIssuerURL),
		arg("oidc-client-id", testClientID),
		arg("oidc-username-claim", "email"),
		arg("oidc-groups-claim", "groups"),
	}
}

const testAuthConfigPath = structuredauth.ConfigPath

// testDexIssuer is the management cluster's Dex as the cluster chart renders
// it from global.controlPlane.oidc.
const testDexIssuer = `  - issuer:
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
`

// testGitHubActionsIssuer is a structuredAuthentication.issuers entry the
// workload cluster's OIDC values cannot express.
const testGitHubActionsIssuer = `  - issuer:
      url: "https://token.actions.githubusercontent.com"
      audiences:
        - "my-mc"
      audienceMatchPolicy: "MatchAny"
    claimValidationRules:
      - 
        
        expression: "claims.repository == 'acme/repo'"
        message: "not acme/repo"
    claimMappings:

      username:
        expression: "'github.com/' + claims.actor"
`

// testOtherIssuer is a second issuer the workload cluster's OIDC values can
// express.
const testOtherIssuer = `  - issuer:
      url: "https://login.example.com/tenant"
      audiences:
        - "first-client"
        - "second-client"
      certificateAuthority: |
        -----BEGIN CERTIFICATE-----
        test
        -----END CERTIFICATE-----
    claimMappings:

      username:
        claim: "email"
        prefix: ""
`

// testSharedAudienceIssuer is an issuer the workload cluster's OIDC values
// can express, with an audience of testOtherIssuer.
const testSharedAudienceIssuer = `  - issuer:
      url: "https://login2.example.com/tenant"
      audiences:
        - "second-client"
    claimMappings:

      username:
        claim: "email"
        prefix: ""
`

// testFutureIssuer is Dex-like, but with a field kubectl-gs does not know.
const testFutureIssuer = `  - issuer:
      url: "https://future.example.com"
      audiences:
        - "future-client"
      requiredClaim: "x"
    claimMappings:

      username:
        claim: "email"
        prefix: ""
`

// authConfig returns an AuthenticationConfiguration with the given issuers,
// in order.
func authConfig(issuers ...string) string {
	return "apiVersion: apiserver.config.k8s.io/v1\nkind: AuthenticationConfiguration\njwt:\n" + strings.Join(issuers, "")
}

// testAuthConfig is the AuthenticationConfiguration the cluster chart renders
// with structured authentication: the legacy Dex issuer first, then the
// structuredAuthentication.issuers.
var testAuthConfig = authConfig(testDexIssuer, testGitHubActionsIssuer)

// structuredKCP returns a management cluster KubeadmControlPlane with
// structured authentication and the given AuthenticationConfiguration.
func structuredKCP(config string) *unstructured.Unstructured {
	return withFiles(newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", structuredAuthArgs()), file(testAuthConfigPath, "", config))
}

// structuredAuthArgs are the API server extra arguments with structured
// authentication: no oidc-* argument.
func structuredAuthArgs() []interface{} {
	return []interface{}{
		arg("audit-log-maxage", "30"),
		arg("authentication-config", testAuthConfigPath),
	}
}

// withFiles sets the KubeadmControlPlane's kubeadmConfigSpec.files.
func withFiles(kcp *unstructured.Unstructured, files ...interface{}) *unstructured.Unstructured {
	spec := kcp.Object["spec"].(map[string]interface{})["kubeadmConfigSpec"].(map[string]interface{})
	spec["files"] = files
	return kcp
}

func file(path, encoding, content string) interface{} {
	f := map[string]interface{}{"path": path, "content": content}
	if encoding != "" {
		f["encoding"] = encoding
	}
	return f
}

var dexValues = OIDCValues{
	IssuerURL:     testIssuerURL,
	ClientID:      testClientID,
	UsernameClaim: "email",
	GroupsClaim:   "groups",
}

func Test_ManagementClusterOIDC(t *testing.T) {
	noMatch := func(ctx context.Context, c client.WithWatch, obj client.Object) error {
		if obj.GetObjectKind().GroupVersionKind().Version == "v1beta2" {
			return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: testGroup, Kind: testKind}, SearchedVersions: []string{"v1beta2"}}
		}
		return nil
	}

	testCases := []struct {
		name           string
		objects        []client.Object
		interceptor    interceptor.Funcs
		mcName         string
		clientID       string
		expectedValues OIDCValues
		expectedError  string
		expectedCause  func(error) bool
	}{
		{
			name:           "case 0: the only control plane in org-giantswarm",
			objects:        []client.Object{newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs())},
			expectedValues: dexValues,
		},
		{
			name: "case 1: the named management cluster among several control planes",
			objects: []client.Object{
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs()),
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "ops", []interface{}{}),
			},
			mcName:         "my-mc",
			expectedValues: dexValues,
		},
		{
			name: "case 2: extra arguments as a v1beta1 map",
			objects: []client.Object{newKubeadmControlPlane("v1beta1", "org-giantswarm", "my-mc", map[string]interface{}{
				"oidc-issuer-url": testIssuerURL,
				"oidc-client-id":  testClientID,
			})},
			mcName: "my-mc",
			interceptor: interceptor.Funcs{Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if err := noMatch(ctx, c, obj); err != nil {
					return err
				}
				return c.Get(ctx, key, obj, opts...)
			}},
			expectedValues: OIDCValues{IssuerURL: testIssuerURL, ClientID: testClientID},
		},
		{
			name: "case 3: no control plane in org-giantswarm",
			objects: []client.Object{
				newKubeadmControlPlane("v1beta2", "org-acme", "workload", dexArgs()),
			},
			expectedError: "no KubeadmControlPlane found in namespace org-giantswarm",
		},
		{
			name:          "case 4: named management cluster not found",
			objects:       []client.Object{newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs())},
			mcName:        "other-mc",
			expectedError: `the kubeconfig context does not point at management cluster "other-mc"`,
		},
		{
			name: "case 5: several control planes and no name",
			objects: []client.Object{
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs()),
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "ops", []interface{}{}),
			},
			expectedError: "name the management cluster with --management-cluster",
		},
		{
			name: "case 6: the API server trusts no OIDC issuer",
			objects: []client.Object{newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", []interface{}{
				arg("oidc-issuer-url", testIssuerURL),
			})},
			expectedError: "trusts no complete OIDC issuer",
		},
		{
			name: "case 7: management cluster unreachable",
			interceptor: interceptor.Funcs{List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return errors.New("dial tcp 10.0.0.1:443: connect: connection refused")
			}},
			expectedError: "the management cluster did not answer: dial tcp 10.0.0.1:443: connect: connection refused",
		},
		{
			name: "case 8: no KubeadmControlPlane served (a workload cluster context)",
			interceptor: interceptor.Funcs{List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: testGroup, Kind: testKind}}
			}},
			expectedError: "serves no KubeadmControlPlane resource",
			// The no-match error stays in the chain for the login hint.
			expectedCause: meta.IsNoMatchError,
		},
		{
			name: "case 9: structured authentication, base64-encoded file",
			objects: []client.Object{withFiles(
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", structuredAuthArgs()),
				file("/etc/kubernetes/policies/audit-policy.yaml", "base64", base64.StdEncoding.EncodeToString([]byte("kind: Policy"))),
				file(testAuthConfigPath, "base64", base64.StdEncoding.EncodeToString([]byte(testAuthConfig))),
			)},
			expectedValues: dexValues,
		},
		{
			name:           "case 10: structured authentication, plain file",
			objects:        []client.Object{structuredKCP(testAuthConfig)},
			expectedValues: dexValues,
		},
		{
			name:          "case 11: structured authentication without the file",
			objects:       []client.Object{newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", structuredAuthArgs())},
			expectedError: "KubeadmControlPlane org-giantswarm/my-mc writes no JWT issuer inline to " + testAuthConfigPath,
		},
		{
			name:          "case 12: structured authentication without a JWT issuer",
			objects:       []client.Object{structuredKCP(authConfig())},
			expectedError: "KubeadmControlPlane org-giantswarm/my-mc writes no JWT issuer inline to " + testAuthConfigPath,
		},
		{
			name:          "case 13: the client ID is the audience of an issuer that cannot be copied only",
			objects:       []client.Object{structuredKCP(testAuthConfig)},
			clientID:      "my-mc",
			expectedError: `no JWT issuer with audience "my-mc" can be copied to the workload cluster's OIDC settings: https://token.actions.githubusercontent.com (claim validation rules, username CEL expression)`,
		},
		{
			name:          "case 14: no issuer can be copied",
			objects:       []client.Object{structuredKCP(authConfig(testGitHubActionsIssuer))},
			expectedError: "no JWT issuer can be copied to the workload cluster's OIDC settings: https://token.actions.githubusercontent.com (claim validation rules, username CEL expression)",
		},
		{
			name: "case 15: oidc-* arguments take precedence over the file",
			objects: []client.Object{withFiles(
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs()),
				file(testAuthConfigPath, "", authConfig(testOtherIssuer)),
			)},
			expectedValues: dexValues,
		},
		{
			name: "case 16: the file without the authentication-config argument",
			// The API server does not load it, e.g. structured authentication
			// enabled below Kubernetes 1.34.
			objects: []client.Object{withFiles(
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", []interface{}{}),
				file(testAuthConfigPath, "", testAuthConfig),
			)},
			expectedError: "sets no oidc-issuer-url or oidc-client-id",
		},
		{
			name:          "case 17: several issuers can be copied and no client ID",
			objects:       []client.Object{structuredKCP(authConfig(testDexIssuer, testGitHubActionsIssuer, testOtherIssuer))},
			expectedError: "2 JWT issuers can be copied to the workload cluster's OIDC settings; select one by its audience with --oidc-client-id: https://dex.my-mc.example.com (audiences dex-k8s-authenticator); https://login.example.com/tenant (audiences first-client, second-client)",
		},
		{
			name:     "case 18: several issuers can be copied, selected by any audience, which becomes the client ID, with its CA",
			objects:  []client.Object{structuredKCP(authConfig(testDexIssuer, testOtherIssuer))},
			clientID: "second-client",
			expectedValues: OIDCValues{
				IssuerURL:     "https://login.example.com/tenant",
				CAPem:         "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n",
				ClientID:      "second-client",
				UsernameClaim: "email",
			},
		},
		{
			name:          "case 19: several issuers can be copied and no audience matches the client ID",
			objects:       []client.Object{structuredKCP(authConfig(testDexIssuer, testOtherIssuer))},
			clientID:      "unknown",
			expectedError: `no JWT issuer has audience "unknown"; 2 can be copied to the workload cluster's OIDC settings, select one by its audience with --oidc-client-id`,
		},
		{
			name: "case 20: a client ID that is no issuer's audience keeps the only one that can be copied",
			// The caller then overrides Dex's client ID with the flag, as
			// with a management cluster that uses the oidc-* arguments.
			objects:        []client.Object{structuredKCP(testAuthConfig)},
			clientID:       "unknown",
			expectedValues: dexValues,
		},
		{
			name:          "case 21: the client ID is the audience of several issuers that can be copied",
			objects:       []client.Object{structuredKCP(authConfig(testDexIssuer, testOtherIssuer, testSharedAudienceIssuer))},
			clientID:      "second-client",
			expectedError: `2 JWT issuers that can be copied to the workload cluster's OIDC settings have audience "second-client": https://login.example.com/tenant (audiences first-client, second-client); https://login2.example.com/tenant (audiences second-client)`,
		},
		{
			name:          "case 22: an issuer with a field kubectl-gs does not know cannot be copied",
			objects:       []client.Object{structuredKCP(authConfig(testFutureIssuer))},
			expectedError: "no JWT issuer can be copied to the workload cluster's OIDC settings: https://future.example.com (unknown fields issuer.requiredClaim)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrlClient := fake.NewClientBuilder().WithObjects(tc.objects...).WithInterceptorFuncs(tc.interceptor).Build()

			values, err := ManagementClusterOIDC(context.Background(), ctrlClient, tc.mcName, tc.clientID)
			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got %v", tc.expectedError, err)
				}
				if !IsManagementClusterOIDC(err) {
					t.Fatalf("expected ErrManagementClusterOIDC, got %#v", err)
				}
				if tc.expectedCause != nil && !tc.expectedCause(err) {
					t.Fatalf("expected the cause to stay in the error chain, got %#v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(tc.expectedValues, values); diff != "" {
				t.Fatalf("values differ (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_valuesGaps(t *testing.T) {
	email := structuredauth.ClaimMapping{Claim: "email"}

	testCases := []struct {
		name         string
		issuer       structuredauth.Issuer
		expectedGaps []string
	}{
		{
			name:   "case 0: Dex as the cluster chart renders it",
			issuer: structuredauth.Issuer{Audiences: []string{"dex-k8s-authenticator"}, Username: email, Groups: structuredauth.ClaimMapping{Claim: "groups"}},
		},
		{
			name:         "case 1: username claim other than email",
			issuer:       structuredauth.Issuer{Audiences: []string{"c"}, Username: structuredauth.ClaimMapping{Claim: "sub"}},
			expectedGaps: []string{`username claim "sub" instead of email`},
		},
		{
			name: "case 2: everything the values cannot express",
			issuer: structuredauth.Issuer{
				DiscoveryURL:         "https://discovery.example.com",
				EgressSelectorType:   "controlplane",
				Username:             structuredauth.ClaimMapping{Claim: "email", Prefix: "oidc:"},
				Groups:               structuredauth.ClaimMapping{Expression: "claims.roles"},
				MapsUIDOrExtra:       true,
				ClaimValidationRules: 1,
				UserValidationRules:  1,
				UnknownFields:        []string{"futureRules", "issuer.requiredClaim"},
			},
			expectedGaps: []string{"no audience", "discovery URL", "egress selector type", "claim validation rules", "user validation rules", "username prefix", "groups CEL expression", "uid or extra mappings", "unknown fields futureRules, issuer.requiredClaim"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.expectedGaps, valuesGaps(tc.issuer)); diff != "" {
				t.Fatalf("gaps differ (-want +got):\n%s", diff)
			}
		})
	}
}
