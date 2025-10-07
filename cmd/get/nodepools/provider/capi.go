package provider

import (
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

func GetCAPITable(npResource nodepool.Resource) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Cluster Name", Type: "string"},
			{Name: "Age", Type: "string", Format: "date-time"},
			{Name: "Phase", Type: "string"},
			{Name: "Nodes Min/Max", Type: "string"},
			{Name: "Nodes Desired", Type: "integer"},
			{Name: "Nodes Ready", Type: "integer"},
			{Name: "Description", Type: "string"},
		},
	}

	switch n := npResource.(type) {
	case *nodepool.Nodepool:
		if row := getCAPIMachineDeploymentRow(*n); !isEmptyRow(row) {
			table.Rows = append(table.Rows, row)
		}
		if row := getCAPIMachinePoolRow(*n); !isEmptyRow(row) {
			table.Rows = append(table.Rows, row)
		}
	case *nodepool.Collection:
		// Sort ASC by Cluster name.
		sort.Slice(n.Items, func(i, j int) bool {
			var iClusterName, jClusterName string

			if n.Items[i].MachineDeployment != nil && n.Items[i].MachineDeployment.Labels != nil {
				iClusterName = key.ClusterID(n.Items[i].MachineDeployment)
			}
			if n.Items[j].MachineDeployment != nil && n.Items[j].MachineDeployment.Labels != nil {
				jClusterName = key.ClusterID(n.Items[j].MachineDeployment)
			}

			return strings.Compare(iClusterName, jClusterName) > 0
		})
		sort.Slice(n.Items, func(i, j int) bool {
			var iClusterName, jClusterName string

			if n.Items[i].MachinePool != nil && n.Items[i].MachinePool.Labels != nil {
				iClusterName = key.ClusterID(n.Items[i].MachinePool)
			}
			if n.Items[j].MachinePool != nil && n.Items[j].MachinePool.Labels != nil {
				jClusterName = key.ClusterID(n.Items[j].MachinePool)
			}

			return strings.Compare(iClusterName, jClusterName) > 0
		})
		for _, nodePool := range n.Items {
			if row := getCAPIMachineDeploymentRow(nodePool); !isEmptyRow(row) {
				table.Rows = append(table.Rows, row)
			}
			if row := getCAPIMachinePoolRow(nodePool); !isEmptyRow(row) {
				table.Rows = append(table.Rows, row)
			}
		}
	}

	return table
}

func isEmptyRow(row metav1.TableRow) bool {
	return len(row.Cells) == 0
}

func getCAPIMachineDeploymentRow(nodePool nodepool.Nodepool) metav1.TableRow {
	if nodePool.MachineDeployment == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachineDeployment.GetName(),
			key.ClusterID(nodePool.MachineDeployment),
			output.TranslateTimestampSince(nodePool.MachineDeployment.CreationTimestamp),
			getCAPIMachineDeploymentLatestPhase(nodePool),
			nodePool.MachineDeployment.Status.Replicas,
			nodePool.MachineDeployment.Status.Replicas,
			nodePool.MachineDeployment.Status.ReadyReplicas,
			getCAPIMachineDeploymentDescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachineDeployment,
		},
	}
}

func getCAPIMachinePoolRow(nodePool nodepool.Nodepool) metav1.TableRow {
	if nodePool.MachinePool == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachinePool.GetName(),
			key.ClusterID(nodePool.MachinePool),
			output.TranslateTimestampSince(nodePool.MachinePool.CreationTimestamp),
			getCAPIMachinePoolLatestPhase(nodePool),
			// Only CAPA for now.
			getCAPAAutoscaling(nodePool),
			nodePool.MachinePool.Status.Replicas,
			nodePool.MachinePool.Status.ReadyReplicas,
			getCAPIMachinePoolDescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachinePool,
		},
	}
}

func getCAPIMachineDeploymentLatestPhase(nodePool nodepool.Nodepool) string {
	if nodePool.MachineDeployment.Status.Phase != "" {
		return nodePool.MachineDeployment.Status.Phase
	}

	return naValue
}

func getCAPIMachinePoolLatestPhase(nodePool nodepool.Nodepool) string {
	if nodePool.MachinePool.Status.Phase != "" {
		return nodePool.MachinePool.Status.Phase
	}

	return naValue
}

func getCAPIMachineDeploymentDescription(nodePool nodepool.Nodepool) string {
	description := key.MachineDeploymentName(nodePool.MachineDeployment)
	if len(description) < 1 {
		description = naValue
	}

	return description
}

func getCAPIMachinePoolDescription(nodePool nodepool.Nodepool) string {
	description := key.MachinePoolName(nodePool.MachinePool)
	if len(description) < 1 {
		description = naValue
	}

	return description
}
