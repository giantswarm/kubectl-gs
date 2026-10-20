package common

import (
	"context"
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrlClient := fake.NewClientBuilder().WithObjects(tc.objects...).WithInterceptorFuncs(tc.interceptor).Build()

			values, err := ManagementClusterOIDC(context.Background(), ctrlClient, tc.mcName)
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
