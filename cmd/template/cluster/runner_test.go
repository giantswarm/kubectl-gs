package cluster

import (
	"bytes"
	"context"
	goflag "flag"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
	"github.com/giantswarm/kubectl-gs/v2/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeclient"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
// go test ./cmd/template/cluster -run Test_run -update
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
						ServiceAccount: provider.ServiceAccount{
							Email:  "service-account@email",
							Scopes: []string{"scope1", "scope2"},
						},
					},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_gcp.golden",
		},
		{
			name: "case 1: template cluster capa",
			flags: &flag{
				Name:                     "test1",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: provider.AppConfig{
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
				},
				AWS: provider.AWSConfig{
					MachinePool: provider.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					Role:           "default",
					NetworkVPCCIDR: "10.123.0.0/16",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa.golden",
		},
		{
			name: "case 2: template cluster capa",
			flags: &flag{
				Name:                     "test1",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: provider.AppConfig{
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
				},
				AWS: provider.AWSConfig{
					ClusterType: "proxy-private",
					MachinePool: provider.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					Role:           "default",
					NetworkVPCCIDR: "10.123.0.0/16",
					HttpsProxy:     "https://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					HttpProxy:      "http://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					NoProxy:        "test-domain.com",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_2.golden",
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

			k8sClient := kubeclient.FakeK8sClient()
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
				t.Fatalf("no difference from golden file %s expected, got:\n %s", tc.expectedGoldenFile, diff)
			}
		})
	}
}
