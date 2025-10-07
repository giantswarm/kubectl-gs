package nodepools

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
	"github.com/giantswarm/kubectl-gs/v5/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/v5/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v5/test/kubeconfig"
)

// Test_run uses golden files.
//
// go test ./cmd/get/nodepools -run Test_run -update
func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		storage            []runtime.Object
		args               []string
		clusterName        string
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name: "case 0: get nodepools",
			storage: []runtime.Object{
				newcapiMachineDeployment("1sad2", "s921a", "10.5.0", time.Now(), 2, 1),
				newAWSMachineDeployment("1sad2", "s921a", "10.5.0", "test nodepool 3", time.Now(), 1, 3),
				newcapiMachineDeployment("f930q", "s921a", "11.0.0", time.Now(), 6, 6),
				newAWSMachineDeployment("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 5, 8),
			},
			args:               nil,
			expectedGoldenFile: "run_get_nodepools.golden",
		},
		{
			name:               "case 1: get nodepools, with empty storage",
			args:               nil,
			expectedGoldenFile: "run_get_nodepools_empty_storage.golden",
		},
		{
			name: "case 2: get nodepool by name",
			storage: []runtime.Object{
				newcapiMachineDeployment("1sad2", "s921a", "10.5.0", time.Now(), 2, 1),
				newAWSMachineDeployment("1sad2", "s921a", "10.5.0", "test nodepool 3", time.Now(), 1, 3),
				newcapiMachineDeployment("f930q", "s921a", "11.0.0", time.Now(), 6, 6),
				newAWSMachineDeployment("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 5, 8),
			},
			args:               []string{"f930q"},
			expectedGoldenFile: "run_get_nodepool_by_id.golden",
		},
		{
			name:         "case 3: get nodepool by name, with empty storage",
			args:         []string{"f930q"},
			errorMatcher: IsNotFound,
		},
		{
			name: "case 4: get nodepool by name, with no infrastructure ref",
			storage: []runtime.Object{
				newcapiMachineDeployment("1sad2", "s921a", "10.5.0", time.Now(), 2, 1),
				newAWSMachineDeployment("1sad2", "s921a", "10.5.0", "test nodepool 3", time.Now(), 1, 3),
				newcapiMachineDeployment("f930q", "s921a", "11.0.0", time.Now(), 6, 6),
			},
			args:         []string{"f930q"},
			errorMatcher: IsNotFound,
		},
		{
			name: "case 5: get nodepools by cluster name",
			storage: []runtime.Object{
				newcapiMachineDeployment("1sad2", "s921a", "10.5.0", time.Now(), 2, 1),
				newAWSMachineDeployment("1sad2", "s921a", "10.5.0", "test nodepool 3", time.Now(), 1, 3),
				newcapiMachineDeployment("f930q", "s921a", "11.0.0", time.Now(), 6, 6),
				newAWSMachineDeployment("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 5, 8),
				newcapiMachineDeployment("9f012", "29sa0", "9.0.0", time.Now(), 0, 3),
				newAWSMachineDeployment("9f012", "29sa0", "9.0.0", "test nodepool 5", time.Now(), 1, 1),
			},
			args:               nil,
			clusterName:        "s921a",
			expectedGoldenFile: "run_get_nodepool_by_cluster_id.golden",
		},
		{
			name: "case 6: get nodepools by name and cluster name",
			storage: []runtime.Object{
				newcapiMachineDeployment("1sad2", "s921a", "10.5.0", time.Now(), 2, 1),
				newAWSMachineDeployment("1sad2", "s921a", "10.5.0", "test nodepool 3", time.Now(), 1, 3),
				newcapiMachineDeployment("f930q", "s921a", "11.0.0", time.Now(), 6, 6),
				newAWSMachineDeployment("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 5, 8),
				newcapiMachineDeployment("9f012", "29sa0", "9.0.0", time.Now(), 0, 3),
				newAWSMachineDeployment("9f012", "29sa0", "9.0.0", "test nodepool 5", time.Now(), 1, 1),
			},
			args:               []string{"f930q"},
			clusterName:        "s921a",
			expectedGoldenFile: "run_get_nodepool_by_id_and_cluster_id.golden",
		},
		{
			name:               "case 7: get nodepools by cluster name, with empty storage",
			args:               nil,
			clusterName:        "s921a",
			expectedGoldenFile: "run_get_nodepool_by_cluster_id_empty_storage.golden",
		},
		{
			name:         "case 8: get nodepools by name and cluster name, with empty storage",
			args:         []string{"f930q"},
			clusterName:  "s921a",
			errorMatcher: IsNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()
			flag := &flag{
				print:       genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault),
				ClusterName: tc.clusterName,
			}

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				service:      newClusterService(t, tc.storage...),
				flag:         flag,
				stdout:       out,
				provider:     key.ProviderAWS,
			}

			err := runner.run(ctx, nil, tc.args)
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

func newClusterService(t *testing.T, object ...runtime.Object) *nodepool.Service {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	service, err := nodepool.New(nodepool.Config{
		Client: clients.CtrlClient(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	return service
}
