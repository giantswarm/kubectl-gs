package provider

import (
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
)

func GetKVMTable(resource runtime.Object) *metav1.Table {
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
		case *corev1alpha1.KVMClusterConfigList:
			for _, currentCluster := range c.Items {
				table.Rows = append(table.Rows, getKVMClusterConfigRow(&currentCluster))
			}

		case *corev1alpha1.KVMClusterConfig:
			table.Rows = append(table.Rows, getKVMClusterConfigRow(c))

		default:
			continue
		}
	}

	return table
}

func getKVMClusterConfigRow(cr *corev1alpha1.KVMClusterConfig) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			cr.Spec.Guest.ID,
			cr.CreationTimestamp,
			cr.Spec.Guest.ReleaseVersion,
			cr.Spec.Guest.Owner,
			cr.Spec.Guest.Name,
		},
	}
}
