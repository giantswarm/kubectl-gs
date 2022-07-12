package clusters

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
)

// Test_run uses golden files.
//
//  go test ./cmd/get/clusters -run Test_run -update
//
func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		storage            []runtime.Object
		args               []string
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name: "case 0: get clusters",
			storage: []runtime.Object{
				newcapiCluster("1sad2", "default", "10.5.0", "some-org", "test cluster 3", label.ServicePriorityHighest, nil),
				newAWSClusterResource("1sad2", time.Now().Format(time.RFC3339), "10.5.0", "some-org", "test cluster 3", nil),
				newcapiCluster("f930q", "default", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityMedium, nil),
				newAWSClusterResource("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", nil),
			},
			args:               nil,
			expectedGoldenFile: "run_get_clusters.golden",
		},
		{
			name:               "case 1: get clusters, with empty storage",
			storage:            nil,
			args:               nil,
			expectedGoldenFile: "run_get_clusters_empty_storage.golden",
		},
		{
			name: "case 2: get cluster by id",
			storage: []runtime.Object{
				newcapiCluster("1sad2", time.Now().Format(time.RFC3339), "10.5.0", "some-org", "test cluster 3", label.ServicePriorityHighest, nil),
				newAWSClusterResource("1sad2", time.Now().Format(time.RFC3339), "10.5.0", "some-org", "test cluster 3", nil),
				newcapiCluster("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", label.ServicePriorityMedium, nil),
				newAWSClusterResource("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", nil),
			},
			args:               []string{"f930q"},
			expectedGoldenFile: "run_get_cluster_by_id.golden",
		},
		{
			name:         "case 3: get cluster by id, with empty storage",
			storage:      nil,
			args:         []string{"f930q"},
			errorMatcher: IsNotFound,
		},
		{
			name: "case 4: get cluster by id, with no infrastructure cluster",
			storage: []runtime.Object{
				newcapiCluster("1sad2", "default", "10.5.0", "some-org", "test cluster 3", label.ServicePriorityHighest, nil),
				newAWSClusterResource("1sad2", "2021-01-01T15:04:32Z", "10.5.0", "some-org", "test cluster 3", nil),
				newcapiCluster("f930q", "default", "11.0.0", "some-other", "test cluster 3", label.ServicePriorityMedium, nil),
			},
			args:         []string{"f930q"},
			errorMatcher: IsNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()
			flag := &flag{
				print:  genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault),
				config: genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig),
			}
			out := new(bytes.Buffer)

			runner := &runner{
				service:  newClusterService(t, tc.storage...),
				flag:     flag,
				stdout:   out,
				provider: key.ProviderAWS,
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

func newClusterService(t *testing.T, object ...runtime.Object) *cluster.Service {
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

	return cluster.New(cluster.Config{
		Client: clients.CtrlClient(),
	})
}
