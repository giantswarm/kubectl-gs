package provider

import (
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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
		{Name: "Description", Type: "string"},
	}

	table.Rows = make([]metav1.TableRow, 0, len(clusterLists))
	for _, clusterList := range clusterLists {
		switch c := clusterList.(type) {
		case *infrastructurev1alpha2.AWSClusterList:
			for _, currentCluster := range c.Items {
				table.Rows = append(table.Rows, getAWSClusterRow(&currentCluster))
			}

		case *infrastructurev1alpha2.AWSCluster:
			table.Rows = append(table.Rows, getAWSClusterRow(c))

		case *corev1alpha1.AWSClusterConfigList:
			for _, currentCluster := range c.Items {
				table.Rows = append(table.Rows, getAWSClusterConfigRow(&currentCluster))
			}

		case *corev1alpha1.AWSClusterConfig:
			table.Rows = append(table.Rows, getAWSClusterConfigRow(c))

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
			cr.Spec.Cluster.Description,
		},
	}
}

func getAWSClusterConfigRow(cr *corev1alpha1.AWSClusterConfig) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			cr.Spec.Guest.ID,
			cr.Spec.Guest.Name,
		},
	}
}
