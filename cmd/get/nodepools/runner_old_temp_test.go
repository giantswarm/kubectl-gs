package nodepools

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
	"github.com/giantswarm/kubectl-gs/v2/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeconfig"
)

// Test_run uses golden files.
//
// go test ./cmd/get/nodepools -run Test_run -update
func Test_runOldTemp(t *testing.T) {
	suiteStart := time.Now()

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
				newcapiMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", 2, 1),
				newAWSMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 1, 3),
				newcapiMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", 6, 6),
				newAWSMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 5, 8),
			},
			args:               nil,
			expectedGoldenFile: "run_get_nodepools.golden",
		},
		{
			name:               "case 1: get nodepools, with empty storage",
			storage:            nil,
			args:               nil,
			expectedGoldenFile: "run_get_nodepools_empty_storage.golden",
		},
		{
			name: "case 2: get nodepool by name",
			storage: []runtime.Object{
				newcapiMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", 2, 1),
				newAWSMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 1, 3),
				newcapiMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", 6, 6),
				newAWSMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 5, 8),
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
				newcapiMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", 2, 1),
				newAWSMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 1, 3),
				newcapiMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", 6, 6),
			},
			args:         []string{"f930q"},
			errorMatcher: IsNotFound,
		},
		{
			name: "case 5: get nodepools by cluster name",
			storage: []runtime.Object{
				newcapiMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", 2, 1),
				newAWSMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 1, 3),
				newcapiMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", 6, 6),
				newAWSMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 5, 8),
				newcapiMachineDeploymentOT("9f012", "29sa0", time.Now().Format(time.RFC3339), "9.0.0", 0, 3),
				newAWSMachineDeploymentOT("9f012", "29sa0", time.Now().Format(time.RFC3339), "9.0.0", "test nodepool 5", 1, 1),
			},
			args:               nil,
			clusterName:        "s921a",
			expectedGoldenFile: "run_get_nodepool_by_cluster_id.golden",
		},
		{
			name: "case 6: get nodepools by name and cluster name",
			storage: []runtime.Object{
				newcapiMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", 2, 1),
				newAWSMachineDeploymentOT("1sad2", "s921a", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 1, 3),
				newcapiMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", 6, 6),
				newAWSMachineDeploymentOT("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 5, 8),
				newcapiMachineDeploymentOT("9f012", "29sa0", time.Now().Format(time.RFC3339), "9.0.0", 0, 3),
				newAWSMachineDeploymentOT("9f012", "29sa0", time.Now().Format(time.RFC3339), "9.0.0", "test nodepool 5", 1, 1),
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
			testStart := time.Now()

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
				t.Fatalf("suite start %s, test start %s, test end: %s, value not expected, got:\n %s", suiteStart, testStart, time.Now(), diff)
			}
		})
	}
}
