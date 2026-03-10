package clusters

import (
	"bytes"
	goflag "flag"
	"testing"
	"time"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
	"github.com/giantswarm/kubectl-gs/v5/test/goldenfile"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_printOutput uses golden files.
//
// go test ./cmd/get/clusters -run Test_printOutput -update
func Test_printOutput(t *testing.T) {
	creationTime := time.Now().Add(-10 * time.Hour)

	testCases := []struct {
		name               string
		clusterRes         cluster.Resource
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of clusters, with table output",
			clusterRes: newClusterCollection(
				*newTestCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, creationTime),
				*newTestCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, creationTime),
				*newTestCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, creationTime),
				*newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", creationTime),
				*newTestCluster("9f012", "9.0.0", "test", "test cluster 5", "", creationTime),
				*newTestCluster("2f0as", "10.5.0", "random", "test cluster 6", "", creationTime),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newTestCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z")),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newTestCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z")),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of clusters, with name output",
			clusterRes: newClusterCollection(
				*newTestCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z")),
				*newTestCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z")),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_clusters_name_output.golden",
		},
		{
			name:               "case 4: print single cluster, with table output",
			clusterRes:         newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, creationTime),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_cluster_table_output.golden",
		},
		{
			name:               "case 5: print single cluster, with JSON output",
			clusterRes:         newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_cluster_json_output.golden",
		},
		{
			name:               "case 6: print single cluster, with YAML output",
			clusterRes:         newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_cluster_yaml_output.golden",
		},
		{
			name:               "case 7: print single cluster, with name output",
			clusterRes:         newTestCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z")),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_cluster_name_output.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := &flag{
				print: genericclioptions.NewPrintFlags("").WithDefaultOutput(tc.outputType),
			}
			out := new(bytes.Buffer)
			runner := &runner{
				flag:     flag,
				stdout:   out,
				provider: tc.provider,
			}

			err := runner.printOutput(tc.clusterRes)

			if err != nil {
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

func newUnstructuredCluster(name, namespace string, labels, annotations map[string]string, creationDate time.Time) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.x-k8s.io/v1beta1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":              name,
				"namespace":         namespace,
				"creationTimestamp": creationDate.UTC().Format(time.RFC3339),
			},
		},
	}
	if labels != nil {
		obj.SetLabels(labels)
	}
	if annotations != nil {
		obj.SetAnnotations(annotations)
	}
	return obj
}

func newTestCluster(id, release, org, description, servicePriority string, creationDate time.Time) *cluster.Cluster {
	labels := map[string]string{
		label.ReleaseVersion: release,
		label.Organization:   org,
		key.ClusterNameLabel: id,
	}
	if servicePriority != "" {
		labels[label.ServicePriority] = servicePriority
	}
	annotations := map[string]string{
		annotation.ClusterDescription: description,
	}

	u := newUnstructuredCluster(id, "default", labels, annotations, creationDate)
	return &cluster.Cluster{
		Cluster: u,
	}
}

func newClusterCollection(clusters ...cluster.Cluster) *cluster.Collection {
	collection := &cluster.Collection{
		Items: clusters,
	}

	return collection
}

func Test_printNoResourcesOutput(t *testing.T) {
	expected := "No clusters found.\nTo create a cluster, please check\n\n  kubectl gs template cluster --help\n"
	out := new(bytes.Buffer)
	runner := &runner{
		stdout: out,
	}
	runner.printNoResourcesOutput()

	if out.String() != expected {
		t.Fatalf("value not expected, got:\n %s", out.String())
	}
}

func parseCreated(created string) time.Time {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	return parsedCreationDate
}
