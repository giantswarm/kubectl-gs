package cluster

import (
	"bytes"
	"context"
	goflag "flag"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/test/kubeclient"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
//  go test ./cmd/template/cluster -run Test_run -update
//
func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		flags              *flag
		args               []string
		clusterName        string
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name: "case 0: template cluster gcp",
			flags: &flag{
				Name:         "test1",
				Provider:     "gcp",
				Description:  "just a test cluster",
				Region:       "the-region",
				Organization: "test",
				App: provider.AppConfig{
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
				},
				GCP: provider.GCPConfig{
					Project:        "the-project",
					FailureDomains: []string{"failure-domain1-a", "failure-domain1-b"},
					ControlPlane: provider.GCPControlPlane{
						ServiceAccount: provider.ServiceAccount{
							Email:  "service-account@email",
							Scopes: []string{"scope1", "scope2"},
						},
					},
					MachineDeployment: provider.GCPMachineDeployment{
						Name:             "worker1",
						FailureDomain:    "failure-domain2-b",
						InstanceType:     "very-large",
						Replicas:         7,
						RootVolumeSizeGB: 5,
					},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_gcp.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			out := new(bytes.Buffer)
			tc.flags.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatalf("failed to create logger: %s", err.Error())
			}

			runner := &runner{
				flag:   tc.flags,
				logger: logger,
				stdout: out,
			}

			ssoSecret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "sso-secret",
					Namespace: key.GiantswarmNamespace,
					Labels: map[string]string{
						key.RoleLabel: key.SSHSSOPubKeyLabel,
					},
				},
				Data: map[string][]byte{
					"value": []byte("the-sso-secret"),
				},
			}

			k8sClient := kubeclient.FakeK8sClient(ssoSecret)
			err = runner.run(ctx, k8sClient)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			var expectedResult []byte
			{
				gf := goldenfile.New("testdata", tc.expectedGoldenFile)
				if *update {
					err = gf.Update(out.Bytes())
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
					expectedResult = out.Bytes()
				} else {
					expectedResult, err = gf.Read()
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
				}
			}

			diff := cmp.Diff(string(expectedResult), out.String())
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
}
