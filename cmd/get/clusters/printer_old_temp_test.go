package clusters

import (
	"bytes"
	"testing"
	"time"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/google/go-cmp/cmp"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

// Test_printOutput uses golden files.
//
// go test ./cmd/get/clusters -run Test_printOutput -update
func Test_printOutputOldTemp(t *testing.T) {
	testCases := []struct {
		name               string
		clusterRes         cluster.Resource
		created            string
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of AWS clusters, with table output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", time.Now().Format(time.RFC3339), "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAWSCluster("2a03f", time.Now().Format(time.RFC3339), "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", time.Now().Format(time.RFC3339), "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAWSCluster("9f012", time.Now().Format(time.RFC3339), "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", time.Now().Format(time.RFC3339), "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_aws_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of AWS clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_aws_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of AWS clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			created:            "2021-01-02T15:04:32Z",
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_aws_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS clusters, with name output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_aws_clusters_name_output.golden",
		},
		{
			name:               "case 4: print single AWS cluster, with table output",
			clusterRes:         newAWSCluster("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_aws_cluster_table_output.golden",
		},
		{
			name:               "case 5: print single AWS cluster, with JSON output",
			clusterRes:         newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_aws_cluster_json_output.golden",
		},
		{
			name:               "case 6: print single AWS cluster, with YAML output",
			clusterRes:         newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_aws_cluster_yaml_output.golden",
		},
		{
			name:               "case 7: print single AWS cluster, with name output",
			clusterRes:         newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_aws_cluster_name_output.golden",
		},
		{
			name: "case 8: print list of Azure clusters, with table output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", time.Now().Format(time.RFC3339), "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAzureCluster("2a03f", time.Now().Format(time.RFC3339), "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", time.Now().Format(time.RFC3339), "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAzureCluster("9f012", time.Now().Format(time.RFC3339), "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", time.Now().Format(time.RFC3339), "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_azure_clusters_table_output.golden",
		},
		{
			name: "case 9: print list of Azure clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_azure_clusters_json_output.golden",
		},
		{
			name: "case 10: print list of Azure clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_azure_clusters_yaml_output.golden",
		},
		{
			name: "case 11: print list of Azure clusters, with name output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", "", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", "", []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_azure_clusters_name_output.golden",
		},
		{
			name:               "case 12: print single Azure cluster, with table output",
			clusterRes:         newAzureCluster("f930q", time.Now().Format(time.RFC3339), "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_azure_cluster_table_output.golden",
		},
		{
			name:               "case 13: print single Azure cluster, with JSON output",
			clusterRes:         newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_azure_cluster_json_output.golden",
		},
		{
			name:               "case 14: print single Azure cluster, with YAML output",
			clusterRes:         newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_azure_cluster_yaml_output.golden",
		},
		{
			name:               "case 15: print single Azure cluster, with name output",
			clusterRes:         newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_azure_cluster_name_output.golden",
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

func Test_printNoResourcesOutputOldTemp(t *testing.T) {
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
