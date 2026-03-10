package clusters

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
	"github.com/giantswarm/kubectl-gs/v5/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/v5/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v5/test/kubeconfig"
)

// Test_run uses golden files.
//
// go test ./cmd/get/clusters -run Test_run -update
func Test_run(t *testing.T) {
	creationTime := time.Now().Add(-10 * time.Hour)

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
				newRunnerTestCluster("1sad2", "10.5.0", "some-org", "test cluster 3", label.ServicePriorityHighest, creationTime),
				newRunnerTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityMedium, creationTime),
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
				newRunnerTestCluster("1sad2", "10.5.0", "some-org", "test cluster 3", label.ServicePriorityHighest, creationTime),
				newRunnerTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityMedium, creationTime),
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()
			flag := &flag{
				print: genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault),
			}
			out := new(bytes.Buffer)

			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				service:      newClusterService(t, tc.storage...),
				flag:         flag,
				stdout:       out,
				provider:     key.ProviderCAPA,
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

func newRunnerTestCluster(name, release, org, description, servicePriority string, creationDate time.Time) *unstructured.Unstructured {
	labels := map[string]interface{}{
		label.ReleaseVersion: release,
		label.Organization:   org,
		label.Cluster:        name,
		key.ClusterNameLabel: name,
	}
	if servicePriority != "" {
		labels[label.ServicePriority] = servicePriority
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.x-k8s.io/v1beta1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":              name,
				"namespace":         "default",
				"creationTimestamp": creationDate.UTC().Format(time.RFC3339),
				"labels":            labels,
				"annotations": map[string]interface{}{
					annotation.ClusterDescription: description,
				},
			},
		},
	}
	return obj
}

func newClusterService(t *testing.T, object ...runtime.Object) *cluster.Service {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
	})

	return cluster.New(cluster.Config{
		Client: clients.CtrlClient(),
	})
}
