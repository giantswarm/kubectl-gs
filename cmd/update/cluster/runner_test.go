package cluster

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
)

func Test_run(t *testing.T) {
	var testCases = []struct {
		name    string
		storage []runtime.Object
		flags   flag
	}{
		{
			name:    "update cluster",
			storage: []runtime.Object{newCluster("abcd1", "default", "16.0.1"), newAWSCluster("abcd1", "default", "16.0.1")},
			flags:   flag{Name: "abcd1", ReleaseVersion: "16.1.0", ScheduledTime: "30 Jan 22 12:00 UTC", Provider: "aws"},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()

			flag := &tc.flags
			flag.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)
			flag.config = genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)

			out := new(bytes.Buffer)
			runner := &runner{
				flag:    flag,
				stdout:  out,
				service: cluster.NewFakeService(tc.storage),
			}

			err = runner.run(ctx, nil, []string{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func newCluster(name, namespace, targetRelease string) *capiv1alpha3.Cluster {
	c := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha3",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				capiv1alpha3.ClusterLabelName: name,
				label.ReleaseVersion:          "16.0.1",
			},
			Annotations: map[string]string{
				"cluster.giantswarm.io/description": "fake-cluster",
			},
		},
	}

	return c
}

func newAWSCluster(name, namespace, targetRelease string) *infrastructurev1alpha3.AWSCluster {
	c := &infrastructurev1alpha3.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.giantswarm.io/v1alpha3",
			Kind:       "AWSCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster: name,
			},
			Annotations: map[string]string{
				"cluster.giantswarm.io/description": "fake-cluster",
			},
		},
	}

	return c
}
