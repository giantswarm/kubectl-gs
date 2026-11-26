package cluster

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/test/kubeclient"
)

const (
	testIssuerURL = "https://idp.example.com"
	testClientID  = "my-client"
)

func Test_oidcValues(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, []byte("-----BEGIN CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name           string
		flag           *flags.Flag
		expectedValues *common.OIDCValues
		expectedError  bool
	}{
		{
			// No management cluster control plane exists: EKS and AKS never ask.
			name: "case 0: eks renders no OIDC settings",
			flag: &flags.Flag{Provider: key.ProviderEKS},
		},
		{
			name: "case 1: aks renders no OIDC settings",
			flag: &flags.Flag{Provider: key.ProviderAKS, OIDC: common.OIDC{IssuerURL: testIssuerURL, ClientID: testClientID}},
		},
		{
			name: "case 2: an explicit issuer with a CA file",
			flag: &flags.Flag{Provider: key.ProviderCAPA, OIDC: common.OIDC{IssuerURL: testIssuerURL, ClientID: testClientID, CAFile: caFile}},
			expectedValues: &common.OIDCValues{
				IssuerURL:     testIssuerURL,
				CAPem:         "-----BEGIN CERTIFICATE-----\n",
				ClientID:      testClientID,
				UsernameClaim: "email",
				GroupsClaim:   "groups",
			},
		},
		{
			name:          "case 3: a CA file that cannot be read",
			flag:          &flags.Flag{Provider: key.ProviderCAPA, OIDC: common.OIDC{IssuerURL: testIssuerURL, ClientID: testClientID, CAFile: filepath.Join(t.TempDir(), "missing.pem")}},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &runner{flag: tc.flag}

			values, err := r.oidcValues(context.Background(), kubeclient.FakeK8sClient())
			if tc.expectedError {
				if err == nil {
					t.Fatal("expected an error, got none")
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

// controlPlane returns a KubeadmControlPlane in org-giantswarm whose API
// server carries the given extra arguments.
func controlPlane(name string, extraArgs ...interface{}) *unstructured.Unstructured {
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
	kcp.SetAPIVersion("controlplane.cluster.x-k8s.io/v1beta2")
	kcp.SetKind("KubeadmControlPlane")
	kcp.SetNamespace("org-giantswarm")
	kcp.SetName(name)
	return kcp
}

func Test_oidcValues_managementClusterName(t *testing.T) {
	// The management cluster's API server trusts its Dex. The workload
	// cluster's trusts no OIDC issuer, so picking it fails.
	mc := controlPlane("my-mc",
		apiServerArg("oidc-issuer-url", "https://dex.my-mc.example.com"),
		apiServerArg("oidc-client-id", "dex-k8s-authenticator"),
		apiServerArg("oidc-username-claim", "email"),
		apiServerArg("oidc-groups-claim", "groups"),
	)
	wc := controlPlane("my-wc", apiServerArg("audit-log-maxage", "30"))
	metadata := func(name string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "flux-giantswarm", Name: "management-cluster-metadata"},
			Data:       map[string]string{"NAME": name},
		}
	}
	mcValues := &common.OIDCValues{
		IssuerURL:     "https://dex.my-mc.example.com",
		ClientID:      "dex-k8s-authenticator",
		UsernameClaim: "email",
		GroupsClaim:   "groups",
	}

	testCases := []struct {
		name           string
		objects        []runtime.Object
		expectedValues *common.OIDCValues
		expectedError  []string
	}{
		{
			name:           "case 0: the metadata ConfigMap picks the management cluster among workload clusters",
			objects:        []runtime.Object{mc, wc, metadata("my-mc")},
			expectedValues: mcValues,
		},
		{
			name:           "case 1: without the metadata ConfigMap, the only control plane is the management cluster's",
			objects:        []runtime.Object{mc},
			expectedValues: mcValues,
		},
		{
			name:          "case 2: without the metadata ConfigMap, several control planes are ambiguous",
			objects:       []runtime.Object{mc, wc},
			expectedError: []string{"flux-giantswarm/management-cluster-metadata", "2 KubeadmControlPlanes found", "--management-cluster"},
		},
		{
			name:          "case 3: the metadata ConfigMap names a cluster without a control plane",
			objects:       []runtime.Object{mc, wc, metadata("other")},
			expectedError: []string{`flux-giantswarm/management-cluster-metadata names management cluster "other"`, "no KubeadmControlPlane org-giantswarm/other found"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &runner{
				flag:   &flags.Flag{Provider: key.ProviderCAPA},
				stderr: io.Discard,
			}

			values, err := r.oidcValues(context.Background(), kubeclient.FakeK8sClient(tc.objects...))
			if tc.expectedError != nil {
				if !common.IsManagementClusterOIDC(err) {
					t.Fatalf("expected a management cluster OIDC error, got %v", err)
				}
				for _, want := range tc.expectedError {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("expected error to contain %q, got %q", want, err)
					}
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

func Test_oidcValues_noKubeContext(t *testing.T) {
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(kubeconfig, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	configFlags := genericclioptions.NewConfigFlags(false)
	configFlags.KubeConfig = &kubeconfig
	var getter genericclioptions.RESTClientGetter = configFlags

	r := &runner{
		commonConfig: &commonconfig.CommonConfig{ConfigFlags: &getter},
		flag:         &flags.Flag{Provider: key.ProviderCAPA},
	}

	_, err := r.oidcValues(context.Background(), kubeclient.FakeK8sClient())
	if !common.IsManagementClusterOIDC(err) {
		t.Fatalf("expected a management cluster OIDC error, got %v", err)
	}
	for _, want := range []string{"no kubeconfig context is selected", "--disable-oidc"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err)
		}
	}
}

func Test_reportManagementClusterOIDC(t *testing.T) {
	var stderr strings.Builder
	r := &runner{stderr: &stderr}

	r.reportManagementClusterOIDC(common.ManagementClusterIssuer{
		Values:            common.OIDCValues{IssuerURL: "https://dex.my-mc.example.com", ClientID: "typo"},
		ManagementCluster: "my-mc",
		Source:            "/etc/kubernetes/policies/structured-auth.yaml",
		ClientIDOverride:  []string{"dex-k8s-authenticator"},
		Skipped:           []string{"https://token.actions.githubusercontent.com"},
	})

	want := "OIDC: the workload cluster trusts https://dex.my-mc.example.com with client ID typo, the Dex of management cluster my-mc, read from /etc/kubernetes/policies/structured-auth.yaml.\n" +
		"OIDC: --oidc-client-id typo is none of the client IDs the management cluster's Dex accepts (dex-k8s-authenticator); using it as given.\n" +
		"OIDC: not copied, as only the management cluster's Dex is: https://token.actions.githubusercontent.com.\n"
	if diff := cmp.Diff(want, stderr.String()); diff != "" {
		t.Fatalf("report differs (-want +got):\n%s", diff)
	}
}
