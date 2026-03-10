package nodepools

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
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
	"github.com/giantswarm/kubectl-gs/v5/test/goldenfile"
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
			name: "case 0: print list of CAPA nodepools, with table output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", time.Now(), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", time.Now(), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", time.Now(), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", time.Now(), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", time.Now(), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", time.Now(), 2, 5, -1, -1),
			),
			provider:           key.ProviderDefault,
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_capa_nodepools_table_output.golden",
		},
		{
			name: "case 1: print list of CAPA nodepools, with JSON output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderDefault,
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_capa_nodepools_json_output.golden",
		},
		{
			name: "case 2: print list of CAPA nodepools, with YAML output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderDefault,
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_capa_nodepools_yaml_output.golden",
		},
		{
			name: "case 3: print list of CAPA nodepools, with name output",
			np: newNodePoolCollection(
				*newCAPANodePool("1sad2", "s921a", "test nodepool 1", parseCreated("2021-01-02T15:04:32Z"), 1, 3, -1, -1),
				*newCAPANodePool("2a03f", "3a0d1", "test nodepool 2", parseCreated("2021-01-02T15:04:32Z"), 3, 10, -1, -1),
				*newCAPANodePool("asd29", "s0a10", "test nodepool 3", parseCreated("2021-01-02T15:04:32Z"), 10, 10, 10, 10),
				*newCAPANodePool("f930q", "s921a", "test nodepool 4", parseCreated("2021-01-02T15:04:32Z"), 3, 3, -1, -1),
				*newCAPANodePool("9f012", "29sa0", "test nodepool 5", parseCreated("2021-01-02T15:04:32Z"), 0, 3, 1, 1),
				*newCAPANodePool("2f0as", "s00sn", "test nodepool 6", parseCreated("2021-01-02T15:04:32Z"), 2, 5, -1, -1),
			),
			provider:           key.ProviderDefault,
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

func newUnstructuredMachinePool(name, clusterName, release, description string, creationDate time.Time, nodesDesired, nodesReady int) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.x-k8s.io/v1beta1",
			"kind":       "MachinePool",
			"metadata": map[string]interface{}{
				"name":              name,
				"namespace":         "org-giantswarm",
				"creationTimestamp": creationDate.UTC().Format(time.RFC3339),
				"labels": map[string]interface{}{
					label.Cluster:        clusterName,
					label.MachinePool:    name,
					label.ReleaseVersion: release,
					label.Organization:   "giantswarm",
					key.ClusterNameLabel: clusterName,
				},
				"annotations": map[string]interface{}{
					annotation.MachinePoolName: description,
				},
			},
			"status": map[string]interface{}{
				"replicas":      int64(nodesDesired),
				"readyReplicas": int64(nodesReady),
			},
		},
	}
	return obj
}

func newUnstructuredCAPAMachinePool(name, clusterName string, minSize, maxSize int) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "infrastructure.cluster.x-k8s.io/v1beta2",
			"kind":       "AWSMachinePool",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "org-giantswarm",
				"labels": map[string]interface{}{
					label.MachinePool:    name,
					label.Organization:   "giantswarm",
					key.ClusterNameLabel: clusterName,
				},
			},
			"spec": map[string]interface{}{
				"minSize": int64(minSize),
				"maxSize": int64(maxSize),
			},
		},
	}
	return obj
}

func newCAPANodePool(name, clusterName, description string, creationDate time.Time, nodesMin, nodesMax, nodesDesired, nodesReady int) *nodepool.Nodepool {
	mp := newUnstructuredMachinePool(name, clusterName, "", description, creationDate, nodesDesired, nodesReady)
	capaMP := newUnstructuredCAPAMachinePool(name, clusterName, nodesMin, nodesMax)

	return &nodepool.Nodepool{
		MachinePool:     mp,
		CAPAMachinePool: capaMP,
	}
}

func newNodePoolCollection(nps ...nodepool.Nodepool) *nodepool.Collection {
	return &nodepool.Collection{
		Items: nps,
	}
}

func parseCreated(created string) time.Time {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	return parsedCreationDate
}
