package cluster

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
