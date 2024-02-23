package provider

import (
	"fmt"
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
		table.Rows = append(table.Rows, getCAPINodePoolRow(*n, capabilities))
	case *nodepool.Collection:
		// Sort ASC by Cluster name.
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
			table.Rows = append(table.Rows, getCAPINodePoolRow(nodePool, capabilities))
		}
	}

	return table
}

func getCAPINodePoolRow(
	nodePool nodepool.Nodepool,
	capabilities *feature.Service,
) metav1.TableRow {
	if nodePool.MachinePool == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachinePool.GetName(),
			key.ClusterID(nodePool.MachinePool),
			output.TranslateTimestampSince(nodePool.MachinePool.CreationTimestamp),
			getCAPILatestCondition(nodePool, capabilities),
			getCAPIAutoscaling(nodePool, capabilities),
			nodePool.MachinePool.Status.Replicas,
			nodePool.MachinePool.Status.ReadyReplicas,
			getCAPIDescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachinePool,
		},
	}
}

func getCAPILatestCondition(nodePool nodepool.Nodepool, capabilities *feature.Service) string {
	if len(nodePool.MachinePool.Status.Conditions) > 0 {
		return formatCondition(string(nodePool.MachinePool.Status.Conditions[0].Type))
	}

	return naValue
}

func getCAPIAutoscaling(nodePool nodepool.Nodepool, capabilities *feature.Service) string {
	switch capabilities.Provider() {
	case key.ProviderCAPA:
		return getCAPAAutoscaling(nodePool)
	case key.ProviderCAPZ:
		return getCAPZAutoscaling(nodePool)
	default:
		return naValue
	}
}

func getCAPAAutoscaling(nodePool nodepool.Nodepool) string {
	minScaling := nodePool.CAPAMachinePool.Spec.MinSize
	maxScaling := nodePool.CAPAMachinePool.Spec.MaxSize

	return fmt.Sprintf("%d/%d", minScaling, maxScaling)
}

func getCAPZAutoscaling(nodePool nodepool.Nodepool) string {
	minScaling, maxScaling := key.MachinePoolScaling(nodePool.MachinePool)
	if minScaling >= 0 && maxScaling >= 0 {
		return fmt.Sprintf("%d/%d", minScaling, maxScaling)
	}
	return naValue
}

func getCAPIDescription(nodePool nodepool.Nodepool) string {
	description := key.MachinePoolName(nodePool.MachinePool)
	if len(description) < 1 {
		description = naValue
	}

	return description
}
