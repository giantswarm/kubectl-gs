package nodepool

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
)

func Test_getWorkloadCluster(t *testing.T) {
	testCases := []struct {
		name         string
		wcNamespace  string
		wcName       string
		wcProvider   string
		storage      []runtime.Object
		errorMatcher func(error) bool
	}{
		{
			name:        "case 0: with existing cluster",
			wcNamespace: "org-test",
			wcName:      "test-wc",
			wcProvider:  "aws",
			storage: []runtime.Object{
				newCAPICluster("org-test", "test-wc"),
				newAWSCluster("org-test", "test-wc"),
			},
		},
		{
			name:        "case 1: with non-existing cluster",
			wcNamespace: "org-test",
			wcName:      "some-test-wc",
			wcProvider:  "aws",
			storage: []runtime.Object{
				newCAPICluster("org-test", "test-wc"),
				newAWSCluster("org-test", "test-wc"),
			},
			errorMatcher: IsClusterNotFound,
		},
		{
			name:        "case 2: with no given namespace, existing cluster",
			wcNamespace: "",
			wcName:      "test-wc",
			wcProvider:  "aws",
			storage: []runtime.Object{
				newCAPICluster("org-test", "test-wc"),
				newAWSCluster("org-test", "test-wc"),
				newCAPICluster("org-test", "test-wc2"),
				newAWSCluster("org-test", "test-wc2"),
			},
		},
		{
			name:        "case 3: with no given namespace, non-existing cluster",
			wcNamespace: "",
			wcName:      "test-wc",
			wcProvider:  "aws",
			storage: []runtime.Object{
				newCAPICluster("org-test2", "test-wc3"),
				newAWSCluster("org-test2", "test-wc3"),
				newCAPICluster("org-test", "test-wc2"),
				newAWSCluster("org-test", "test-wc2"),
			},
			errorMatcher: IsClusterNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			clusterService := cluster.NewFakeService(tc.storage)

			wc, err := getWorkloadCluster(ctx, clusterService, tc.wcNamespace, tc.wcName, tc.wcProvider)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if wc == nil {
				t.Fatalf("unexpected nil cluster")
			}
		})
	}
}

func newCAPICluster(namespace, name string) *capiv1alpha3.Cluster {
	c := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.x-k8s.io/v1alpha3",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels: map[string]string{
				label.ReleaseVersion:          "1.2.3",
				label.Organization:            "some-org",
				capiv1alpha3.ClusterLabelName: name,
			},
		},
	}

	return c
}

func newAWSCluster(namespace, name string) *infrastructurev1alpha3.AWSCluster {
	c := &infrastructurev1alpha3.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels: map[string]string{
				label.ReleaseVersion:          "1.2.3",
				label.Organization:            "some-org",
				label.Cluster:                 name,
				capiv1alpha3.ClusterLabelName: name,
			},
		},
	}

	return c
}
