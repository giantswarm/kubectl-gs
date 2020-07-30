package clusters

import (
	"bytes"
	goflag "flag"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/pkg/annotation"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_printOutput uses golden files.
//
//  go test ./cmd/get/clusters -run Test_printOutput -update
//
func Test_printOutput(t *testing.T) {
	testCases := []struct {
		name               string
		cr                 runtime.Object
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of AWS clusters, with table output",
			cr: newAWSClusterList(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_aws_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of AWS clusters, with JSON output",
			cr: newAWSClusterList(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_aws_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of AWS clusters, with YAML output",
			cr: newAWSClusterList(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_aws_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS clusters, with name output",
			cr: newAWSClusterList(
				*newAWSCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAWSCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_aws_clusters_name_output.golden",
		},
		{
			name:               "case 4: print single AWS cluster, with table output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_aws_cluster_table_output.golden",
		},
		{
			name:               "case 5: print single AWS cluster, with JSON output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_aws_cluster_json_output.golden",
		},
		{
			name:               "case 6: print single AWS cluster, with YAML output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_aws_cluster_yaml_output.golden",
		},
		{
			name:               "case 7: print single AWS cluster, with name output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_aws_cluster_name_output.golden",
		},
		{
			name: "case 8: print list of Azure clusters, with table output",
			cr: newAzureClusterList(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_azure_clusters_table_output.golden",
		},
		{
			name: "case 9: print list of Azure clusters, with JSON output",
			cr: newAzureClusterList(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_azure_clusters_json_output.golden",
		},
		{
			name: "case 10: print list of Azure clusters, with YAML output",
			cr: newAzureClusterList(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_azure_clusters_yaml_output.golden",
		},
		{
			name: "case 11: print list of Azure clusters, with name output",
			cr: newAzureClusterList(
				*newAzureCluster("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				*newAzureCluster("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				*newAzureCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_azure_clusters_name_output.golden",
		},
		{
			name:               "case 12: print single Azure cluster, with table output",
			cr:                 newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_azure_cluster_table_output.golden",
		},
		{
			name:               "case 13: print single Azure cluster, with JSON output",
			cr:                 newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_azure_cluster_json_output.golden",
		},
		{
			name:               "case 14: print single Azure cluster, with YAML output",
			cr:                 newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_azure_cluster_yaml_output.golden",
		},
		{
			name:               "case 15: print single Azure cluster, with name output",
			cr:                 newAzureCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
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

			err := runner.printOutput(tc.cr)
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

func newAWSClusterList(clusters ...infrastructurev1alpha2.AWSCluster) *infrastructurev1alpha2.AWSClusterList {
	clusterList := &infrastructurev1alpha2.AWSClusterList{}

	clusterList.Items = clusters
	clusterList.APIVersion = "v1"
	clusterList.Kind = "List"

	return clusterList
}

func newAWSCluster(id, created, release, org, description string, conditions []string) *infrastructurev1alpha2.AWSCluster {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	c := &infrastructurev1alpha2.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.ReleaseVersion: release,
				label.Organization:   org,
			},
		},
		Spec: infrastructurev1alpha2.AWSClusterSpec{
			Cluster: infrastructurev1alpha2.AWSClusterSpecCluster{
				Description: description,
			},
		},
	}
	for _, condition := range conditions {
		c.Status.Cluster.Conditions = append(c.Status.Cluster.Conditions, infrastructurev1alpha2.CommonClusterStatusCondition{
			Condition: condition,
		})
	}

	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   infrastructurev1alpha2.SchemeGroupVersion.Group,
		Version: infrastructurev1alpha2.SchemeGroupVersion.Version,
		Kind:    infrastructurev1alpha2.NewAWSClusterTypeMeta().Kind,
	})

	return c
}

func newAzureClusterList(clusters ...capiv1alpha3.Cluster) *capiv1alpha3.ClusterList {
	clusterList := &capiv1alpha3.ClusterList{}

	clusterList.Items = clusters
	clusterList.APIVersion = "v1"
	clusterList.Kind = "List"

	return clusterList
}

func newAzureCluster(id, created, release, org, description string, conditions []string) *capiv1alpha3.Cluster {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	c := &capiv1alpha3.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.ReleaseVersion: release,
				label.Organization:   org,
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: description,
			},
		},
	}

	if len(conditions) > 0 && conditions[0] == "Created" {
		c.Status.InfrastructureReady = true
		c.Status.ControlPlaneInitialized = true
		c.Status.ControlPlaneReady = true
	}

	{
		c.APIVersion = "v1alpha3"
		c.Kind = "Cluster"
	}

	return c
}

func Test_printNoResourcesOutput(t *testing.T) {
	expected := `No clusters found.
To create a cluster, please check

  kgs create cluster --help
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
