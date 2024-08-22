package clusters

import (
	"bytes"
	goflag "flag"
	"testing"
	"time"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/k8smetadata/pkg/label"

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
	testCases := []struct {
		name               string
		clusterRes         cluster.Resource
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of AWS clusters, with table output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, time.Now(), nil),
				*newAWSCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", time.Now(), nil),
				*newAWSCluster("9f012", "9.0.0", "test", "test cluster 5", "", time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "10.5.0", "random", "test cluster 6", "", time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_aws_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of AWS clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_aws_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of AWS clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_aws_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS clusters, with name output",
			clusterRes: newClusterCollection(
				*newAWSCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAWSCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAWSCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAWSCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_aws_clusters_name_output.golden",
		},
		{
			name:               "case 4: print single AWS cluster, with table output",
			clusterRes:         newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_aws_cluster_table_output.golden",
		},
		{
			name:               "case 5: print single AWS cluster, with JSON output",
			clusterRes:         newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_aws_cluster_json_output.golden",
		},
		{
			name:               "case 6: print single AWS cluster, with YAML output",
			clusterRes:         newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_aws_cluster_yaml_output.golden",
		},
		{
			name:               "case 7: print single AWS cluster, with name output",
			clusterRes:         newAWSCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_aws_cluster_name_output.golden",
		},
		{
			name: "case 8: print list of Azure clusters, with table output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, time.Now(), nil),
				*newAzureCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", time.Now(), nil),
				*newAzureCluster("9f012", "9.0.0", "test", "test cluster 5", "", time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "10.5.0", "random", "test cluster 6", "", time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_azure_clusters_table_output.golden",
		},
		{
			name: "case 9: print list of Azure clusters, with JSON output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_azure_clusters_json_output.golden",
		},
		{
			name: "case 10: print list of Azure clusters, with YAML output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_azure_clusters_yaml_output.golden",
		},
		{
			name: "case 11: print list of Azure clusters, with name output",
			clusterRes: newClusterCollection(
				*newAzureCluster("1sad2", "12.0.0", "test", "test cluster 1", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("2a03f", "11.0.0", "test", "test cluster 2", label.ServicePriorityMedium, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
				*newAzureCluster("asd29", "10.5.0", "test", "test cluster 3", label.ServicePriorityLowest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated, infrastructurev1alpha3.ClusterStatusConditionCreating}),
				*newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", "", parseCreated("2021-01-02T15:04:32Z"), nil),
				*newAzureCluster("9f012", "9.0.0", "test", "test cluster 5", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting}),
				*newAzureCluster("2f0as", "10.5.0", "random", "test cluster 6", "", parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionDeleting, infrastructurev1alpha3.ClusterStatusConditionCreated}),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_azure_clusters_name_output.golden",
		},
		{
			name:               "case 12: print single Azure cluster, with table output",
			clusterRes:         newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, time.Now(), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_azure_cluster_table_output.golden",
		},
		{
			name:               "case 13: print single Azure cluster, with JSON output",
			clusterRes:         newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_azure_cluster_json_output.golden",
		},
		{
			name:               "case 14: print single Azure cluster, with YAML output",
			clusterRes:         newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_azure_cluster_yaml_output.golden",
		},
		{
			name:               "case 15: print single Azure cluster, with name output",
			clusterRes:         newAzureCluster("f930q", "11.0.0", "some-other", "test cluster 4", label.ServicePriorityHighest, parseCreated("2021-01-02T15:04:32Z"), []string{infrastructurev1alpha3.ClusterStatusConditionCreated}),
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

func newAWSClusterResource(id, release, org, description string, creationDate time.Time, conditions []string) *infrastructurev1alpha3.AWSCluster {
	c := &infrastructurev1alpha3.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(creationDate),
			Labels: map[string]string{
				label.ReleaseVersion:  release,
				label.Organization:    org,
				label.Cluster:         id,
				capi.ClusterNameLabel: id,
			},
		},
		Spec: infrastructurev1alpha3.AWSClusterSpec{
			Cluster: infrastructurev1alpha3.AWSClusterSpecCluster{
				Description: description,
			},
		},
	}
	for _, condition := range conditions {
		c.Status.Cluster.Conditions = append(c.Status.Cluster.Conditions, infrastructurev1alpha3.CommonClusterStatusCondition{
			Condition: condition,
		})
	}

	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   infrastructurev1alpha3.SchemeGroupVersion.Group,
		Version: infrastructurev1alpha3.SchemeGroupVersion.Version,
		Kind:    infrastructurev1alpha3.NewAWSClusterTypeMeta().Kind,
	})

	return c
}

func newAWSCluster(id, release, org, description, servicePriority string, creationDate time.Time, conditions []string) *cluster.Cluster {
	awsCluster := newAWSClusterResource(id, release, org, description, creationDate, conditions)
	capiCluster := newcapiCluster(id, release, org, description, servicePriority, parseCreated("default"), conditions)

	c := &cluster.Cluster{
		AWSCluster: awsCluster,
		Cluster:    capiCluster,
	}

	return c
}

func newAzureClusterResource(id, namespace string) *capz.AzureCluster {
	c := &capz.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: namespace,
			Labels: map[string]string{
				capi.ClusterNameLabel: id,
			}},
	}

	return c
}

func newAzureCluster(id, release, org, description, servicePriority string, creationDate time.Time, conditions []string) *cluster.Cluster {
	azureCluster := newAzureClusterResource(id, "default")
	capiCluster := newcapiCluster(id, release, org, description, servicePriority, creationDate, conditions)

	c := &cluster.Cluster{
		AzureCluster: azureCluster,
		Cluster:      capiCluster,
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
