package clusters

import (
	"bytes"
	goflag "flag"
	"testing"
	"time"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capi "sigs.k8s.io/cluster-api/api/core/v1beta1"

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
	// Use a fixed time in the past to avoid flaky tests due to timing races
	// where AGE shows "0s" locally but "1s" in CI.
	creationTime := time.Now().Add(-10 * time.Hour)

	testCases := []struct {
		name               string
		clusterRes         cluster.Resource
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of CAPI clusters, with table output",
			clusterRes: newClusterCollection(
				*newCAPICluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, creationTime, nil),
				*newCAPICluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, creationTime, nil),
				*newCAPICluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, creationTime, nil),
				*newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", "", creationTime, nil),
				*newCAPICluster("9f012", "9.0.0", "test", "test cluster 5", "", creationTime, nil),
				*newCAPICluster("2f0as", "10.5.0", "random", "test cluster 6", "", creationTime, nil),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_capi_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of CAPI clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newCAPICluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), nil),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_capi_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of CAPI clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newCAPICluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), nil),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_capi_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of CAPI clusters, with name output",
			clusterRes: newClusterCollection(
				*newCAPICluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newCAPICluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), nil),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_capi_clusters_name_output.golden",
		},
		{
			name:               "case 4: print single CAPI cluster, with table output",
			clusterRes:         newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, creationTime, nil),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_capi_cluster_table_output.golden",
		},
		{
			name:               "case 5: print single CAPI cluster, with JSON output",
			clusterRes:         newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_capi_cluster_json_output.golden",
		},
		{
			name:               "case 6: print single CAPI cluster, with YAML output",
			clusterRes:         newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_capi_cluster_yaml_output.golden",
		},
		{
			name:               "case 7: print single CAPI cluster, with name output",
			clusterRes:         newCAPICluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_capi_cluster_name_output.golden",
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

func newcapiCluster(id, release, org, description, servicePriority string, creationDate time.Time, conditions []string) *capi.Cluster {
	c := &capi.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(creationDate),
			Labels: map[string]string{
				label.ReleaseVersion:  release,
				label.Organization:    org,
				capi.ClusterNameLabel: id,
				label.ServicePriority: servicePriority,
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: description,
			},
		},
	}

	resConditions := make([]capi.Condition, 0, len(conditions))
	for _, condition := range conditions {
		resConditions = append(resConditions, capi.Condition{
			Type:   capi.ConditionType(condition),
			Status: "True",
		})
	}
	c.SetConditions(resConditions)

	return c
}

func newCAPICluster(id, release, org, description, servicePriority string, creationDate time.Time, conditions []string) *cluster.Cluster {
	capiCluster := newcapiCluster(id, release, org, description, servicePriority, creationDate, conditions)

	c := &cluster.Cluster{
		Cluster: capiCluster,
	}

	return c
}

func newClusterCollection(clusters ...cluster.Cluster) *cluster.Collection {
	collection := &cluster.Collection{
		Items: clusters,
	}

	return collection
}

func Test_printNoResourcesOutput(t *testing.T) {
	expected := `No clusters found.
To create a cluster, please check

  kubectl gs template cluster --help
`
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
