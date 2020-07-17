package clusters

import (
	"bytes"
	goflag "flag"
	"fmt"
	"testing"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
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
			cr: newCommonClusterList([]runtime.Object{
				newAWSV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAWSV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_list_of_aws_clusters_table_output.golden",
		},
		{
			name: "case 1: print list of AWS clusters, with JSON output",
			cr: newCommonClusterList([]runtime.Object{
				newAWSV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAWSV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_list_of_aws_clusters_json_output.golden",
		},
		{
			name: "case 2: print list of AWS clusters, with YAML output",
			cr: newCommonClusterList([]runtime.Object{
				newAWSV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAWSV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_list_of_aws_clusters_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS clusters, with name output",
			cr: newCommonClusterList([]runtime.Object{
				newAWSV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAWSV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
				newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionCreated, infrastructurev1alpha2.ClusterStatusConditionCreating}),
				newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", nil),
				newAWSCluster("9f012", "2021-01-02T15:04:32Z", "9.0.0", "test", "test cluster 5", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAWSCluster("2f0as", "2021-01-02T15:04:32Z", "10.5.0", "random", "test cluster 6", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_list_of_aws_clusters_name_output.golden",
		},
		{
			name: "case 4: print list of Azure clusters, with table output",
			cr: newCommonClusterList([]runtime.Object{
				newAzureV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAzureV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAzure,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_list_of_azure_clusters_table_output.golden",
		},
		{
			name: "case 5: print list of Azure clusters, with JSON output",
			cr: newCommonClusterList([]runtime.Object{
				newAzureV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAzureV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAzure,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_list_of_azure_clusters_json_output.golden",
		},
		{
			name: "case 6: print list of Azure clusters, with YAML output",
			cr: newCommonClusterList([]runtime.Object{
				newAzureV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAzureV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAzure,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_list_of_azure_clusters_yaml_output.golden",
		},
		{
			name: "case 7: print list of Azure clusters, with name output",
			cr: newCommonClusterList([]runtime.Object{
				newAzureV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newAzureV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderAzure,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_list_of_azure_clusters_name_output.golden",
		},
		{
			name: "case 8: print list of KVM clusters, with table output",
			cr: newCommonClusterList([]runtime.Object{
				newKVMV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newKVMV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_list_of_kvm_clusters_table_output.golden",
		},
		{
			name: "case 9: print list of KVM clusters, with JSON output",
			cr: newCommonClusterList([]runtime.Object{
				newKVMV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newKVMV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_list_of_kvm_clusters_json_output.golden",
		},
		{
			name: "case 10: print list of KVM clusters, with YAML output",
			cr: newCommonClusterList([]runtime.Object{
				newKVMV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newKVMV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_list_of_kvm_clusters_yaml_output.golden",
		},
		{
			name: "case 11: print list of KVM clusters, with name output",
			cr: newCommonClusterList([]runtime.Object{
				newKVMV4ClusterList("1sad2", "2021-01-02T15:04:32Z", "12.0.0", "test", "test cluster 1", nil),
				newKVMV4ClusterList("2a03f", "2021-01-02T15:04:32Z", "11.0.0", "test", "test cluster 2", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
				newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_list_of_kvm_clusters_name_output.golden",
		},
		{
			name:               "case 12: print single v4 AWS cluster, with table output",
			cr:                 newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_single_aws_v4_cluster_table_output.golden",
		},
		{
			name:               "case 13: print single v4 AWS cluster, with JSON output",
			cr:                 newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_single_aws_v4_cluster_json_output.golden",
		},
		{
			name:               "case 14: print single v4 AWS cluster, with YAML output",
			cr:                 newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_single_aws_v4_cluster_yaml_output.golden",
		},
		{
			name:               "case 15: print single v4 AWS cluster, with name output",
			cr:                 newAWSV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_single_aws_v4_cluster_name_output.golden",
		},
		{
			name:               "case 16: print single v5 AWS cluster, with table output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_single_aws_v5_cluster_table_output.golden",
		},
		{
			name:               "case 17: print single v5 AWS cluster, with JSON output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_single_aws_v5_cluster_json_output.golden",
		},
		{
			name:               "case 18: print single v5 AWS cluster, with YAML output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_single_aws_v5_cluster_yaml_output.golden",
		},
		{
			name:               "case 19: print single v5 AWS cluster, with name output",
			cr:                 newAWSCluster("f930q", "2021-01-02T15:04:32Z", "11.0.0", "some-other", "test cluster 4", []string{infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderAWS,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_single_aws_v5_cluster_name_output.golden",
		},
		{
			name:               "case 20: print single v4 Azure cluster, with table output",
			cr:                 newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", nil),
			provider:           key.ProviderAzure,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_single_azure_v4_cluster_table_output.golden",
		},
		{
			name:               "case 21: print single v4 Azure cluster, with JSON output",
			cr:                 newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", nil),
			provider:           key.ProviderAzure,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_single_azure_v4_cluster_json_output.golden",
		},
		{
			name:               "case 22: print single v4 Azure cluster, with YAML output",
			cr:                 newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", nil),
			provider:           key.ProviderAzure,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_single_azure_v4_cluster_yaml_output.golden",
		},
		{
			name:               "case 23: print single v4 Azure cluster, with name output",
			cr:                 newAzureV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", nil),
			provider:           key.ProviderAzure,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_single_azure_v4_cluster_name_output.golden",
		},
		{
			name:               "case 24: print single v4 KVM cluster, with table output",
			cr:                 newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputDefault,
			expectedGoldenFile: "print_single_kvm_v4_cluster_table_output.golden",
		},
		{
			name:               "case 25: print single v4 KVM cluster, with JSON output",
			cr:                 newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputJSON,
			expectedGoldenFile: "print_single_kvm_v4_cluster_json_output.golden",
		},
		{
			name:               "case 26: print single v4 KVM cluster, with YAML output",
			cr:                 newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputYAML,
			expectedGoldenFile: "print_single_kvm_v4_cluster_yaml_output.golden",
		},
		{
			name:               "case 27: print single v4 KVM cluster, with name output",
			cr:                 newKVMV4ClusterList("asd29", "2021-01-02T15:04:32Z", "10.5.0", "test", "test cluster 3", []string{infrastructurev1alpha2.ClusterStatusConditionDeleting, infrastructurev1alpha2.ClusterStatusConditionCreated}),
			provider:           key.ProviderKVM,
			outputType:         output.OutputName,
			expectedGoldenFile: "print_single_kvm_v4_cluster_name_output.golden",
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

			gf := goldenfile.New("testdata", tc.expectedGoldenFile)
			expectedResult, err := gf.Read()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if *update {
				err = gf.Update(out.Bytes())
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			diff := cmp.Diff(string(expectedResult), out.String())
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
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

func newAWSV4ClusterList(id, created, release, org, description string, conditions []string) *cluster.V4ClusterList {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)

	cc := &corev1alpha1.AWSClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-aws-cluster-config", id),
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: corev1alpha1.AWSClusterConfigSpec{
			Guest: corev1alpha1.AWSClusterConfigSpecGuest{
				ClusterGuestConfig: corev1alpha1.ClusterGuestConfig{
					ID:             id,
					ReleaseVersion: release,
					Owner:          org,
					Name:           description,
				},
			},
		},
	}
	cc.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "AWSClusterConfig",
	})

	c := &providerv1alpha1.AWSConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: providerv1alpha1.AWSConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: id,
			},
		},
	}
	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "AWSConfig",
	})
	for _, condition := range conditions {
		c.Status.Cluster.Conditions = append(c.Status.Cluster.Conditions, providerv1alpha1.StatusClusterCondition{
			Type: condition,
		})
	}

	newCluster := &cluster.V4ClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: []runtime.Object{cc, c},
	}

	return newCluster
}

func newAzureV4ClusterList(id, created, release, org, description string, conditions []string) *cluster.V4ClusterList {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)

	cc := &corev1alpha1.AzureClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-azure-cluster-config", id),
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: corev1alpha1.AzureClusterConfigSpec{
			Guest: corev1alpha1.AzureClusterConfigSpecGuest{
				ClusterGuestConfig: corev1alpha1.ClusterGuestConfig{
					ID:             id,
					ReleaseVersion: release,
					Owner:          org,
					Name:           description,
				},
			},
		},
	}
	cc.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "AzureClusterConfig",
	})

	c := &providerv1alpha1.AzureConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: providerv1alpha1.AzureConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: id,
			},
		},
	}
	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "AzureConfig",
	})
	for _, condition := range conditions {
		c.Status.Cluster.Conditions = append(c.Status.Cluster.Conditions, providerv1alpha1.StatusClusterCondition{
			Type: condition,
		})
	}

	newCluster := &cluster.V4ClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: []runtime.Object{cc, c},
	}

	return newCluster
}

func newKVMV4ClusterList(id, created, release, org, description string, conditions []string) *cluster.V4ClusterList {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)

	cc := &corev1alpha1.KVMClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-kvm-cluster-config", id),
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: corev1alpha1.KVMClusterConfigSpec{
			Guest: corev1alpha1.KVMClusterConfigSpecGuest{
				ClusterGuestConfig: corev1alpha1.ClusterGuestConfig{
					ID:             id,
					ReleaseVersion: release,
					Owner:          org,
					Name:           description,
				},
			},
		},
	}
	cc.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "KVMClusterConfig",
	})

	c := &providerv1alpha1.KVMConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              id,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
		},
		Spec: providerv1alpha1.KVMConfigSpec{
			Cluster: providerv1alpha1.Cluster{
				ID: id,
			},
		},
	}
	c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   providerv1alpha1.SchemeGroupVersion.Group,
		Version: providerv1alpha1.SchemeGroupVersion.Version,
		Kind:    "KVMConfig",
	})
	for _, condition := range conditions {
		c.Status.Cluster.Conditions = append(c.Status.Cluster.Conditions, providerv1alpha1.StatusClusterCondition{
			Type: condition,
		})
	}

	newCluster := &cluster.V4ClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: []runtime.Object{cc, c},
	}

	return newCluster
}

func newCommonClusterList(lists []runtime.Object) *cluster.CommonClusterList {
	list := &cluster.CommonClusterList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "List",
		},
		Items: lists,
	}

	return list
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
	err := runner.printNoResourcesOutput()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if out.String() != expected {
		t.Fatalf("value not expected, got:\n %s", out.String())
	}
}
