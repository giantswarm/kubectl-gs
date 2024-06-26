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
	capaexp "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v3/internal/key"
	"github.com/giantswarm/kubectl-gs/v3/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v3/pkg/output"
	"github.com/giantswarm/kubectl-gs/v3/test/goldenfile"
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
				*newAWSNodePool("1sad2", "s921a", "12.0.0", "test nodepool 1", time.Now(), 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "11.0.0", "test nodepool 2", time.Now(), 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "10.5.0", "test nodepool 3", time.Now(), 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "9.0.0", "test nodepool 5", time.Now(), 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "10.5.0", "test nodepool 6", time.Now(), 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_aws_nodepools_table_output.golden",
		},
		{
			name: "case 1: print list of AWS nodepools, with JSON output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "12.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "11.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "10.5.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "9.0.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "10.5.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_aws_nodepools_json_output.golden",
		},
		{
			name: "case 2: print list of AWS nodepools, with YAML output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "12.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "11.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "10.5.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "9.0.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "10.5.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_aws_nodepools_yaml_output.golden",
		},
		{
			name: "case 3: print list of AWS nodepools, with name output",
			np: newNodePoolCollection(
				*newAWSNodePool("1sad2", "s921a", "12.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, 2, 2),
				*newAWSNodePool("2a03f", "3a0d1", "11.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, 5, 2),
				*newAWSNodePool("asd29", "s0a10", "10.5.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
				*newAWSNodePool("9f012", "29sa0", "9.0.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAWSNodePool("2f0as", "s00sn", "10.5.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, 5, 5),
			),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_aws_nodepools_name_output.golden",
		},
		{
			name:               "case 4: print single AWS nodepool, with table output",
			np:                 newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", time.Now(), 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_aws_nodepool_table_output.golden",
		},
		{
			name:               "case 5: print single AWS nodepool, with JSON output",
			np:                 newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_aws_nodepool_json_output.golden",
		},
		{
			name:               "case 6: print single AWS nodepool, with YAML output",
			np:                 newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_aws_nodepool_yaml_output.golden",
		},
		{
			name:               "case 7: print single AWS nodepool, with name output",
			np:                 newAWSNodePool("f930q", "s921a", "11.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, 3, 1),
			provider:           key.ProviderAWS,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_aws_nodepool_name_output.golden",
		},
		{
			name: "case 8: print list of Azure nodepools, with table output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "13.0.0", "test nodepool 1", time.Now(), 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "13.0.0", "test nodepool 2", time.Now(), 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "13.2.0", "test nodepool 3", time.Now(), 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", time.Now(), 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "13.2.0", "test nodepool 5", time.Now(), 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "13.1.0", "test nodepool 6", time.Now(), 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_azure_nodepools_table_output.golden",
		},
		{
			name: "case 9: print list of Azure nodepools, with JSON output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "13.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "13.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "13.2.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "13.2.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "13.1.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_azure_nodepools_json_output.golden",
		},
		{
			name: "case 10: print list of Azure nodepools, with YAML output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "13.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "13.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "13.2.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "13.2.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "13.1.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_azure_nodepools_yaml_output.golden",
		},
		{
			name: "case 11: print list of Azure nodepools, with name output",
			np: newNodePoolCollection(
				*newAzureNodePool("1sad2", "s921a", "13.0.0", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newAzureNodePool("2a03f", "3a0d1", "13.0.0", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newAzureNodePool("asd29", "s0a10", "13.2.0", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newAzureNodePool("9f012", "29sa0", "13.2.0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newAzureNodePool("2f0as", "s00sn", "13.1.0", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_azure_nodepools_name_output.golden",
		},
		{
			name:               "case 12: print single Azure nodepool, with table output",
			np:                 newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", time.Now(), 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_azure_nodepool_table_output.golden",
		},
		{
			name:               "case 13: print single Azure nodepool, with JSON output",
			np:                 newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_azure_nodepool_json_output.golden",
		},
		{
			name:               "case 14: print single Azure nodepool, with YAML output",
			np:                 newAzureNodePool("f930q", "s921a", "13.0.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_azure_nodepool_yaml_output.golden",
		},
		{
			name:               "case 15: print single Azure nodepool, with name output",
			np:                 newAzureNodePool("f930q", "s921a", "13.2.0", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
			provider:           key.ProviderAzure,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_azure_nodepool_name_output.golden",
		},
		{
			name: "case 16: print list of CAPA nodepools, with table output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", time.Now(), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", time.Now(), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", time.Now(), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", time.Now(), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", time.Now(), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", time.Now(), 2, 5, -1, -1),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_capa_nodepools_table_output.golden",
		},
		{
			name: "case 17: print list of CAPA nodepools, with JSON output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_capa_nodepools_json_output.golden",
		},
		{
			name: "case 18: print list of CAPA nodepools, with YAML output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_capa_nodepools_yaml_output.golden",
		},
		{
			name: "case 19: print list of CAPA nodepools, with name output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderCAPA,
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_capa_nodepools_name_output.golden",
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

func newAWSMachineDeployment(name, clusterName, release, description string, creationDate time.Time, nodesMin, nodesMax int) *infrastructurev1alpha3.AWSMachineDeployment {
	n := &infrastructurev1alpha3.AWSMachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(creationDate),
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

func newcapiMachineDeployment(name, clusterName, release string, creationDate time.Time, nodesDesired, nodesReady int) *capi.MachineDeployment {
	n := &capi.MachineDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "MachineDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(creationDate),
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

func newAWSNodePool(name, clusterName, release, description string, creationDate time.Time, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	awsMD := newAWSMachineDeployment(name, clusterName, release, description, creationDate, nodesMin, nodesMax)
	md := newcapiMachineDeployment(name, clusterName, release, creationDate, nodesDesired, nodesReady)

	np := &nodepool.Nodepool{
		MachineDeployment:    md,
		AWSMachineDeployment: awsMD,
	}

	return np
}

func newAzureMachinePool(name, clusterName, release string, creationDate time.Time) *capzexp.AzureMachinePool {
	n := &capzexp.AzureMachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureMachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "org-giantswarm",
			CreationTimestamp: metav1.NewTime(creationDate),
			Labels: map[string]string{
				label.MachinePool:     name,
				label.ReleaseVersion:  release,
				label.Organization:    "giantswarm",
				capi.ClusterNameLabel: clusterName,
			},
		},
	}

	return n
}

func newCAPAexpMachinePool(name, clusterName, description string, creationDate time.Time) *capaexp.AWSMachinePool {
	n := &capaexp.AWSMachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "exp.infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSMachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "org-giantswarm",
			CreationTimestamp: metav1.NewTime(creationDate),
			Labels: map[string]string{
				label.MachinePool:     name,
				label.Organization:    "giantswarm",
				capi.ClusterNameLabel: clusterName,
			},
		},
	}

	return n
}

func newCAPIexpMachinePool(name, clusterName, release, description string, creationDate time.Time, nodesDesired, nodesReady, nodesMin, nodesMax int) *capiexp.MachinePool {
	n := &capiexp.MachinePool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "exp.cluster.x-k8s.io/v1beta1",
			Kind:       "MachinePool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "org-giantswarm",
			CreationTimestamp: metav1.NewTime(creationDate),
			Labels: map[string]string{
				label.Cluster:         clusterName,
				label.MachinePool:     name,
				label.ReleaseVersion:  release,
				label.Organization:    "giantswarm",
				capi.ClusterNameLabel: clusterName,
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

func newCAPANodePool(name, clusterName, description string, creationDate time.Time, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	mp := newCAPIexpMachinePool(name, clusterName, "", description, creationDate, nodesMin, nodesMax, nodesDesired, nodesReady)
	capaMP := newCAPAexpMachinePool(name, clusterName, description, creationDate)

	np := &nodepool.Nodepool{
		MachinePool:     mp,
		CAPAMachinePool: capaMP,
	}

	return np
}
func newAzureNodePool(name, clusterName, release, description string, creationDate time.Time, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	azureMP := newAzureMachinePool(name, clusterName, release, creationDate)
	mp := newCAPIexpMachinePool(name, clusterName, release, description, creationDate, nodesMin, nodesMax, nodesDesired, nodesReady)

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

func parseCreated(created string) time.Time {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	return parsedCreationDate
}
