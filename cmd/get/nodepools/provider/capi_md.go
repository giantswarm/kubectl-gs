package provider

import (
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v2/internal/feature"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
)

func GetCAPITable(npResource nodepool.Resource, capabilities *feature.Service) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Cluster Name", Type: "string"},
			{Name: "Age", Type: "string", Format: "date-time"},
			{Name: "Condition", Type: "string"},
			{Name: "Nodes Min/Max", Type: "string"},
			{Name: "Nodes Desired", Type: "integer"},
			{Name: "Nodes Ready", Type: "integer"},
			{Name: "Description", Type: "string"},
		},
	}

	switch n := npResource.(type) {
	case *nodepool.Nodepool:
		table.Rows = append(table.Rows, getCAPINodePoolRow(*n))
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
		for _, nodePool := range n.Items {
			table.Rows = append(table.Rows, getCAPINodePoolRow(nodePool))
		}
	}

	return table
}

func getCAPINodePoolRow(
	nodePool nodepool.Nodepool,
) metav1.TableRow {
	if nodePool.MachineDeployment == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachineDeployment.GetName(),
			key.ClusterID(nodePool.MachineDeployment),
			output.TranslateTimestampSince(nodePool.MachineDeployment.CreationTimestamp),
			getCAPILatestCondition(nodePool),
			nodePool.MachineDeployment.Status.Replicas,
			nodePool.MachineDeployment.Status.Replicas,
			nodePool.MachineDeployment.Status.ReadyReplicas,
			getCAPIDescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachineDeployment,
		},
	}
}

func getCAPILatestCondition(nodePool nodepool.Nodepool) string {
	if len(nodePool.MachineDeployment.Status.Conditions) > 0 {
		return formatCondition(string(nodePool.MachineDeployment.Status.Conditions[0].Type))
	}

	return naValue
}

func getCAPIDescription(nodePool nodepool.Nodepool) string {
	description := key.MachineDeploymentName(nodePool.MachineDeployment)
	if len(description) < 1 {
		description = naValue
	}

	return description
}
