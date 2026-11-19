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

// testDexMultiAudienceIssuer is the management cluster's Dex accepting two
// client IDs.
const testDexMultiAudienceIssuer = `  - issuer:
      url: "https://dex.my-mc.example.com/"
      audiences:
        - "kubernetes"
        - "dex-k8s-authenticator"
    claimMappings:

      username:
        claim: "email"
        prefix: ""
      groups:
        claim: "groups"
        prefix: ""
`

// testDexPrefixedIssuer is the management cluster's Dex with claim prefixes,
// which the workload cluster's OIDC values do not copy.
const testDexPrefixedIssuer = `  - issuer:
      url: "https://dex.my-mc.example.com"
      audiences:
        - "dex-k8s-authenticator"
    claimMappings:

      username:
        claim: "email"
        prefix: "dex:"
      groups:
        claim: "groups"
        prefix: "dex:"
`

// testDexFutureIssuer is the management cluster's Dex with a field
// kubectl-gs does not know.
const testDexFutureIssuer = `  - issuer:
      url: "https://dex.my-mc.example.com"
      audiences:
        - "dex-k8s-authenticator"
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
	return withFiles(withCertSANs(newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", structuredAuthArgs())), file(testAuthConfigPath, "", config))
}

// testCertSANs are the API server certificate SANs the cluster chart renders
// for management cluster my-mc with base domain example.com.
var testCertSANs = []interface{}{"localhost", "127.0.0.1", "api.my-mc.example.com", "apiserver.my-mc.example.com"}

// withCertSANs sets the API server certificate SANs, by default testCertSANs.
func withCertSANs(kcp *unstructured.Unstructured, sans ...interface{}) *unstructured.Unstructured {
	if sans == nil {
		sans = testCertSANs
	}
	apiServer := kcp.Object["spec"].(map[string]interface{})["kubeadmConfigSpec"].(map[string]interface{})["clusterConfiguration"].(map[string]interface{})["apiServer"].(map[string]interface{})
	apiServer["certSANs"] = sans
	return kcp
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
		name             string
		objects          []client.Object
		interceptor      interceptor.Funcs
		mcName           string
		clientID         string
		expectedValues   OIDCValues
		expectedOverride []string
		expectedSkipped  []string
		expectedError    string
		expectedCause    func(error) bool
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
				withCertSANs(newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", structuredAuthArgs())),
				file("/etc/kubernetes/policies/audit-policy.yaml", "base64", base64.StdEncoding.EncodeToString([]byte("kind: Policy"))),
				file(testAuthConfigPath, "base64", base64.StdEncoding.EncodeToString([]byte(testAuthConfig))),
			)},
			expectedValues:  dexValues,
			expectedSkipped: []string{"https://token.actions.githubusercontent.com"},
		},
		{
			name:            "case 10: structured authentication, plain file",
			objects:         []client.Object{structuredKCP(testAuthConfig)},
			expectedValues:  dexValues,
			expectedSkipped: []string{"https://token.actions.githubusercontent.com"},
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
			name:          "case 13: the client ID belongs to another issuer, which is not copied",
			objects:       []client.Object{structuredKCP(testAuthConfig)},
			clientID:      "my-mc",
			expectedError: `client ID "my-mc" belongs to https://token.actions.githubusercontent.com, not to the management cluster's Dex, https://dex.my-mc.example.com, whose client IDs are dex-k8s-authenticator`,
		},
		{
			name:          "case 14: no issuer is the management cluster's Dex",
			objects:       []client.Object{structuredKCP(authConfig(testGitHubActionsIssuer, testOtherIssuer))},
			expectedError: "no JWT issuer is the management cluster's Dex, https://dex.my-mc.example.com. The file lists:\n  - https://token.actions.githubusercontent.com\n  - https://login.example.com/tenant",
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
			name:            "case 17: Dex among other issuers that could be copied, without a client ID",
			objects:         []client.Object{structuredKCP(authConfig(testGitHubActionsIssuer, testOtherIssuer, testDexIssuer))},
			expectedValues:  dexValues,
			expectedSkipped: []string{"https://token.actions.githubusercontent.com", "https://login.example.com/tenant"},
		},
		{
			name:          "case 18: Dex cannot be copied, another issuer is never copied instead",
			objects:       []client.Object{structuredKCP(authConfig(testDexPrefixedIssuer, testOtherIssuer))},
			expectedError: "the management cluster's Dex, https://dex.my-mc.example.com, cannot be copied to the workload cluster's OIDC settings: it sets username prefix, groups prefix",
		},
		{
			name:             "case 19: a client ID that is no issuer's audience replaces Dex's",
			objects:          []client.Object{structuredKCP(authConfig(testDexIssuer, testOtherIssuer))},
			clientID:         "unknown",
			expectedValues:   OIDCValues{IssuerURL: testIssuerURL, ClientID: "unknown", UsernameClaim: "email", GroupsClaim: "groups"},
			expectedOverride: []string{testClientID},
			expectedSkipped:  []string{"https://login.example.com/tenant"},
		},
		{
			name:          "case 20: Dex accepts several client IDs and none is given",
			objects:       []client.Object{structuredKCP(authConfig(testDexMultiAudienceIssuer))},
			expectedError: "the management cluster's Dex, https://dex.my-mc.example.com/, accepts several client IDs: kubernetes, dex-k8s-authenticator. Select one with --oidc-client-id",
		},
		{
			name:     "case 21: the client ID selects one of the several Dex accepts",
			objects:  []client.Object{structuredKCP(authConfig(testDexMultiAudienceIssuer))},
			clientID: testClientID,
			expectedValues: OIDCValues{
				IssuerURL:     "https://dex.my-mc.example.com/",
				ClientID:      testClientID,
				UsernameClaim: "email",
				GroupsClaim:   "groups",
			},
		},
		{
			name:          "case 22: Dex with a field kubectl-gs does not know cannot be copied",
			objects:       []client.Object{structuredKCP(authConfig(testDexFutureIssuer))},
			expectedError: "it sets unknown fields issuer.requiredClaim",
		},
		{
			name:          "case 23: the certificate SANs do not tell the management cluster's domain",
			objects:       []client.Object{withCertSANs(structuredKCP(testAuthConfig), "localhost", "api.other.example.com")},
			expectedError: `cannot tell the URL of management cluster "my-mc"'s Dex`,
		},
		{
			name: "case 24: Dex's CA is copied",
			objects: []client.Object{withCertSANs(withFiles(
				newKubeadmControlPlane("v1beta2", "org-giantswarm", "login", structuredAuthArgs()),
				file(testAuthConfigPath, "", authConfig(strings.Replace(testOtherIssuer, "https://login.example.com/tenant", "https://dex.login.example.com", 1))),
			), "api.login.example.com")},
			clientID: "first-client",
			expectedValues: OIDCValues{
				IssuerURL:     "https://dex.login.example.com",
				CAPem:         "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n",
				ClientID:      "first-client",
				UsernameClaim: "email",
			},
		},
		{
			name:             "case 25: with oidc-* arguments, a different client ID replaces the management cluster's",
			objects:          []client.Object{newKubeadmControlPlane("v1beta2", "org-giantswarm", "my-mc", dexArgs())},
			clientID:         "other",
			expectedValues:   OIDCValues{IssuerURL: testIssuerURL, ClientID: "other", UsernameClaim: "email", GroupsClaim: "groups"},
			expectedOverride: []string{testClientID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrlClient := fake.NewClientBuilder().WithObjects(tc.objects...).WithInterceptorFuncs(tc.interceptor).Build()

			result, err := ManagementClusterOIDC(context.Background(), ctrlClient, tc.mcName, tc.clientID)
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

			if diff := cmp.Diff(tc.expectedValues, result.Values); diff != "" {
				t.Fatalf("values differ (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.expectedOverride, result.ClientIDOverride); diff != "" {
				t.Fatalf("client ID override differs (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.expectedSkipped, result.Skipped); diff != "" {
				t.Fatalf("skipped issuers differ (-want +got):\n%s", diff)
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
