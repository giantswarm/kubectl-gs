package provider

import (
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
)

func GetAWSTable(resource runtime.Object) *metav1.Table {
	var clusterLists []runtime.Object
	{
		switch c := resource.(type) {
		case *cluster.CommonClusterList:
			clusterLists = c.Items
		default:
			clusterLists = []runtime.Object{resource}
		}
	}

	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
		{Name: "Created", Type: "string", Format: "date-time"},
		{Name: "Release", Type: "string"},
		{Name: "Organization", Type: "string"},
		{Name: "Description", Type: "string"},
	}

	table.Rows = make([]metav1.TableRow, 0, len(clusterLists))
	for _, clusterList := range clusterLists {
		switch c := clusterList.(type) {
		case *infrastructurev1alpha2.AWSCluster:
			table.Rows = append(table.Rows, getAWSClusterRow(c))

		case *cluster.V4ClusterList:
			table.Rows = append(table.Rows, getAWSV4ClusterListRow(c))

		default:
			continue
		}
	}

	return table
}

func getAWSClusterRow(cr *infrastructurev1alpha2.AWSCluster) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			cr.GetName(),
			cr.CreationTimestamp.UTC(),
			cr.Labels[label.ReleaseVersion],
			cr.Labels[label.Organization],
			cr.Spec.Cluster.Description,
		},
	}
}

func getAWSV4ClusterListRow(res *cluster.V4ClusterList) metav1.TableRow {
	clusterConfig := res.Items[0].(*corev1alpha1.AWSClusterConfig)
	config := res.Items[1].(*providerv1alpha1.AWSConfig)

	return metav1.TableRow{
		Cells: []interface{}{
			config.Spec.Cluster.ID,
			config.GetCreationTimestamp().UTC(),
			clusterConfig.Spec.Guest.ReleaseVersion,
			clusterConfig.Spec.Guest.Owner,
			clusterConfig.Spec.Guest.Name,
		},
	}
}
