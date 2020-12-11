package provider

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetAWSTable(resource runtime.Object) *metav1.Table {
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
	}

	switch n := resource.(type) {
	case *infrastructurev1alpha2.AWSClusterList:
		for _, nodePool := range n.Items {
			table.Rows = append(table.Rows, getAWSNodePoolRow(&nodePool))
		}

	case *infrastructurev1alpha2.AWSCluster:
		table.Rows = append(table.Rows, getAWSNodePoolRow(n))
	}

	return table
}

func getAWSNodePoolRow(res *infrastructurev1alpha2.AWSCluster) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			res.GetName(),
		},
		Object: runtime.RawExtension{
			Object: res,
		},
	}
}
