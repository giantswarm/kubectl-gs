package nodepools

import (
	"bytes"
	goflag "flag"
	"fmt"
	"testing"
	"time"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_printOutput uses golden files.
//
//	go test ./cmd/get/nodepools -run Test_printOutput -update
func Test_printOutput(t *testing.T) {
	testCases := []struct {
		name               string
		np                 nodepool.Resource
		provider           string
		outputType         string
		expectedGoldenFile string
	}{
		{
			name: "case 0: print list of AWS nodepools, with table output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", time.Now().Format(time.RFC3339), "12.0.0", "test nodepool 1", 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 2", 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 3", 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", time.Now().Format(time.RFC3339), "9.0.0", "test nodepool 5", 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", time.Now().Format(time.RFC3339), "10.5.0", "test nodepool 6", 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_aws_nodepools_table_output.golden",
		},
		{
			name: "case 1: print list of AWS nodepools, with JSON output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "12.0.0", "test nodepool 1", 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 2", 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 3", 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "9.0.0", "test nodepool 5", 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 6", 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_aws_nodepools_json_output.golden",
		},
		{
			name: "case 2: print list of AWS nodepools, with YAML output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "12.0.0", "test nodepool 1", 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 2", 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 3", 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "9.0.0", "test nodepool 5", 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 6", 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_aws_nodepools_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS nodepools, with name output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "12.0.0", "test nodepool 1", 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 2", 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 3", 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "9.0.0", "test nodepool 5", 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "10.5.0", "test nodepool 6", 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_aws_nodepools_name_output.golden",
		},
		{
			name:               "case 4: print single AWS nodepool, with table output",
			np:                 newAWSNodePool("f930q", "s921a", time.Now().Format(time.RFC3339), "11.0.0", "test nodepool 4", 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_aws_nodepool_table_output.golden",
		},
		{
			name:               "case 5: print single AWS nodepool, with JSON output",
			np:                 newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_aws_nodepool_json_output.golden",
		},
		{
			name:               "case 6: print single AWS nodepool, with YAML output",
			np:                 newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_aws_nodepool_yaml_output.golden",
		},
		{
			name:               "case 7: print single AWS nodepool, with name output",
			np:                 newAWSNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "11.0.0", "test nodepool 4", 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_aws_nodepool_name_output.golden",
		},
		{
			name: "case 8: print list of Azure nodepools, with table output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", time.Now().Format(time.RFC3339), "13.0.0", "test nodepool 1", 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", time.Now().Format(time.RFC3339), "13.0.0", "test nodepool 2", 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", time.Now().Format(time.RFC3339), "13.2.0", "test nodepool 3", 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", time.Now().Format(time.RFC3339), "13.0.0", "test nodepool 4", 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", time.Now().Format(time.RFC3339), "13.2.0", "test nodepool 5", 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", time.Now().Format(time.RFC3339), "13.1.0", "test nodepool 6", 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_azure_nodepools_table_output.golden",
		},
		{
			name: "case 9: print list of Azure nodepools, with JSON output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 1", 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 2", 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 3", 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 4", 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 5", 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "13.1.0", "test nodepool 6", 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_azure_nodepools_json_output.golden",
		},
		{
			name: "case 10: print list of Azure nodepools, with YAML output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 1", 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 2", 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 3", 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 4", 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 5", 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "13.1.0", "test nodepool 6", 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_azure_nodepools_yaml_output.golden",
		},
		{
			name: "case 11: print list of Azure nodepools, with name output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 1", 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 2", 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 3", 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 4", 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 5", 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "2021-01-02T15:04:32Z", "13.1.0", "test nodepool 6", 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_azure_nodepools_name_output.golden",
		},
		{
			name:               "case 12: print single Azure nodepool, with table output",
			np:                 newAzureNodePool("f930q", "s921a", time.Now().Format(time.RFC3339), "13.0.0", "test nodepool 4", 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_azure_nodepool_table_output.golden",
		},
		{
			name:               "case 13: print single Azure nodepool, with JSON output",
			np:                 newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 4", 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_azure_nodepool_json_output.golden",
		},
		{
			name:               "case 14: print single Azure nodepool, with YAML output",
			np:                 newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.0.0", "test nodepool 4", 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_azure_nodepool_yaml_output.golden",
		},
		{
			name:               "case 15: print single Azure nodepool, with name output",
			np:                 newAzureNodePool("f930q", "s921a", "2021-01-02T15:04:32Z", "13.2.0", "test nodepool 4", 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_azure_nodepool_name_output.golden",
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

			err := runner.printOutput(tc.np)
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

func newAWSMachineDeployment(name, clusterName, created, release, description string, nodesMin, nodesMax int) *infrastructurev1alpha3.AWSMachineDeployment {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	n := &infrastructurev1alpha3.AWSMachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.MachineDeployment: name,
				label.ReleaseVersion:    release,
				label.Organization:      "giantswarm",
				label.Cluster:           clusterName,
			},
		},
		Spec: infrastructurev1alpha3.AWSMachineDeploymentSpec{
			NodePool: infrastructurev1alpha3.AWSMachineDeploymentSpecNodePool{
				Description: description,
				Scaling: infrastructurev1alpha3.AWSMachineDeploymentSpecNodePoolScaling{
					Min: nodesMin,
					Max: nodesMax,
				},
			},
		},
	}

	n.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   infrastructurev1alpha3.SchemeGroupVersion.Group,
		Version: infrastructurev1alpha3.SchemeGroupVersion.Version,
		Kind:    infrastructurev1alpha3.NewAWSMachineDeploymentTypeMeta().Kind,
	})

	return n
}

func newcapiMachineDeployment(name, clusterName, created, release string, nodesDesired, nodesReady int) *capi.MachineDeployment {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	n := &capi.MachineDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "MachineDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.MachineDeployment: name,
				label.ReleaseVersion:    release,
				label.Organization:      "giantswarm",
				label.Cluster:           clusterName,
			},
		},
		Status: capi.MachineDeploymentStatus{
			Replicas:      int32(nodesDesired),
			ReadyReplicas: int32(nodesReady),
		},
	}

	return n
}

func newAWSNodePool(name, clusterName, created, release, description string, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	awsMD := newAWSMachineDeployment(name, clusterName, created, release, description, nodesMin, nodesMax)
	md := newcapiMachineDeployment(name, clusterName, created, release, nodesDesired, nodesReady)

	np := &nodepool.Nodepool{
		MachineDeployment:    md,
		AWSMachineDeployment: awsMD,
	}

	return np
}

func newAzureMachinePool(name, clusterName, created, release string) *capzexp.AzureMachinePool {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	n := &capzexp.AzureMachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1alpha3",
			Kind:       "AzureMachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "org-giantswarm",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.MachinePool:     name,
				label.ReleaseVersion:  release,
				label.Organization:    "giantswarm",
				capi.ClusterLabelName: clusterName,
			},
		},
	}

	return n
}

func newCAPIexpMachinePool(name, clusterName, created, release, description string, nodesDesired, nodesReady, nodesMin, nodesMax int) *capiexp.MachinePool {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	n := &capiexp.MachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "exp.cluster.x-k8s.io/v1alpha3",
			Kind:       "MachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "org-giantswarm",
			CreationTimestamp: metav1.NewTime(parsedCreationDate),
			Labels: map[string]string{
				label.MachinePool:     name,
				label.ReleaseVersion:  release,
				label.Organization:    "giantswarm",
				capi.ClusterLabelName: clusterName,
			},
			Annotations: map[string]string{
				annotation.NodePoolMinSize: fmt.Sprintf("%d", nodesMin),
				annotation.NodePoolMaxSize: fmt.Sprintf("%d", nodesMax),
				annotation.MachinePoolName: description,
			},
		},
		Status: capiexp.MachinePoolStatus{
			Replicas:      int32(nodesDesired),
			ReadyReplicas: int32(nodesReady),
		},
	}

	return n
}

func newAzureNodePool(name, clusterName, created, release, description string, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	azureMP := newAzureMachinePool(name, clusterName, created, release)
	mp := newCAPIexpMachinePool(name, clusterName, created, release, description, nodesMin, nodesMax, nodesDesired, nodesReady)

	np := &nodepool.Nodepool{
		MachinePool:      mp,
		AzureMachinePool: azureMP,
	}

	return np
}

func newNodePoolCollection(nps ...nodepool.Nodepool) *nodepool.Collection {
	collection := &nodepool.Collection{
		Items: nps,
	}

	return collection
}
