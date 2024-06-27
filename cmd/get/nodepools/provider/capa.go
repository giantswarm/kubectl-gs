package provider

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v3/internal/feature"
	"github.com/giantswarm/kubectl-gs/v3/internal/key"
	"github.com/giantswarm/kubectl-gs/v3/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v3/pkg/output"
)

func GetCAPATable(npResource nodepool.Resource, capabilities *feature.Service) *metav1.Table {
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
		table.Rows = append(table.Rows, getCAPANodePoolRow(*n))
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
			table.Rows = append(table.Rows, getCAPANodePoolRow(nodePool))
		}
	}

	return table
}

func getCAPANodePoolRow(
	nodePool nodepool.Nodepool,
) metav1.TableRow {
	if nodePool.MachinePool == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachinePool.GetName(),
			key.ClusterID(nodePool.MachinePool),
			output.TranslateTimestampSince(nodePool.MachinePool.CreationTimestamp),
			getCAPALatestCondition(nodePool),
			getCAPAAutoscaling(nodePool),
			nodePool.MachinePool.Status.Replicas,
			nodePool.MachinePool.Status.ReadyReplicas,
			getCAPADescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachinePool,
		},
	}
}

func getCAPALatestCondition(nodePool nodepool.Nodepool) string {
	if len(nodePool.MachinePool.Status.Conditions) > 0 {
		return formatCondition(string(nodePool.MachinePool.Status.Conditions[0].Type))
	}

	return naValue
}

func getCAPAAutoscaling(nodePool nodepool.Nodepool) string {
	if nodePool.EKSManagedMachinePool != nil {
		return getEKSManagedAutoscaling(nodePool)
	}

	minScaling := nodePool.CAPAMachinePool.Spec.MinSize
	maxScaling := nodePool.CAPAMachinePool.Spec.MaxSize

	return fmt.Sprintf("%d/%d", minScaling, maxScaling)
}

func getEKSManagedAutoscaling(nodePool nodepool.Nodepool) string {
	minScaling := nodePool.EKSManagedMachinePool.Spec.Scaling.MinSize
	maxScaling := nodePool.EKSManagedMachinePool.Spec.Scaling.MaxSize

	return fmt.Sprintf("%d/%d", *minScaling, *maxScaling)
}

func getCAPADescription(nodePool nodepool.Nodepool) string {
	description := key.MachinePoolName(nodePool.MachinePool)
	if len(description) < 1 {
		description = naValue
	}

	return description
}
