package provider

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func GetAzureTable(resource runtime.Object) *metav1.Table {
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
	}

	switch n := resource.(type) {
	case *capiv1alpha3.ClusterList:
		for _, nodePool := range n.Items {
			table.Rows = append(table.Rows, getAzureNodePoolRow(&nodePool))
		}

	case *capiv1alpha3.Cluster:
		table.Rows = append(table.Rows, getAzureNodePoolRow(n))
	}

	return table
}

func getAzureNodePoolRow(res *capiv1alpha3.Cluster) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			res.GetName(),
		},
		Object: runtime.RawExtension{
			Object: res,
		},
	}
}
