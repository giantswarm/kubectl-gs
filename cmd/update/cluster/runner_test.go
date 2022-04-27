package cluster

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
)

func Test_run(t *testing.T) {
	var testCases = []struct {
		name    string
		storage []runtime.Object
		flags   flag
	}{
		{
			name:    "update cluster with a scheduled time",
			storage: []runtime.Object{newCluster("abcd1", "default", "16.0.1"), newAWSCluster("abcd1", "default", "16.0.1")},
			flags:   flag{Name: "abcd1", ReleaseVersion: "16.1.0", ScheduledTime: "2022-01-01 01:00", Provider: "aws"},
		},
		{
			name:    "update cluster immediately",
			storage: []runtime.Object{newCluster("abcd1", "default", "16.0.1"), newAWSCluster("abcd1", "default", "16.0.1")},
			flags:   flag{Name: "abcd1", ReleaseVersion: "16.1.0", Provider: "aws"},
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
				service: newClusterService(t, tc.storage...),
			}

			err = runner.run(ctx, nil, []string{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func newCluster(name, namespace, targetRelease string) *capiv1beta1.Cluster {
	c := &capiv1beta1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				capiv1beta1.ClusterLabelName: name,
				label.ReleaseVersion:         "16.0.1",
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

func newClusterService(t *testing.T, object ...runtime.Object) *cluster.Service {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	return cluster.New(cluster.Config{
		Client: clients.CtrlClient(),
	})
}
