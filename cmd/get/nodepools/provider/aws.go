package provider

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/nodepool"
)

func GetAWSTable(npResource nodepool.Resource) *metav1.Table {
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
	}

	switch n := npResource.(type) {
	case *nodepool.Nodepool:
		table.Rows = append(table.Rows, getAWSNodePoolRow(*n))
	case *nodepool.Collection:
		for _, nodePool := range n.Items {
			table.Rows = append(table.Rows, getAWSNodePoolRow(nodePool))
		}
	}

	return table
}

func getAWSNodePoolRow(
	nodePool nodepool.Nodepool,
) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachineDeployment.GetName(),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachineDeployment,
		},
	}
}
