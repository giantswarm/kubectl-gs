package provider

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/internal/feature"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/nodepool"
)

func GetAzureTable(npResource nodepool.Resource, capabilities *feature.Service) *metav1.Table {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "ID", Type: "string"},
			{Name: "Cluster ID", Type: "string"},
			{Name: "Created", Type: "string", Format: "date-time"},
			{Name: "Condition", Type: "string"},
			{Name: "Nodes Min/Max", Type: "string"},
			{Name: "Nodes Desired", Type: "integer"},
			{Name: "Nodes Ready", Type: "integer"},
			{Name: "Description", Type: "string"},
		},
	}

	switch n := npResource.(type) {
	case *nodepool.Nodepool:
		table.Rows = append(table.Rows, getAzureNodePoolRow(*n, capabilities))
	case *nodepool.Collection:
		for _, nodePool := range n.Items {
			table.Rows = append(table.Rows, getAzureNodePoolRow(nodePool, capabilities))
		}
	}

	return table
}

func getAzureNodePoolRow(nodePool nodepool.Nodepool, capabilities *feature.Service) metav1.TableRow {
	if nodePool.MachinePool == nil || nodePool.AzureMachinePool == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			nodePool.MachinePool.GetName(),
			nodePool.MachinePool.Labels[capiv1alpha3.ClusterLabelName],
			nodePool.MachinePool.CreationTimestamp.UTC(),
			getAzureLatestCondition(nodePool, capabilities),
			getAzureAutoscaling(nodePool, capabilities),
			nodePool.MachinePool.Status.Replicas,
			nodePool.MachinePool.Status.ReadyReplicas,
			getAzureDescription(nodePool),
		},
		Object: runtime.RawExtension{
			Object: nodePool.MachinePool,
		},
	}
}

func getAzureLatestCondition(nodePool nodepool.Nodepool, capabilities *feature.Service) string {
	releaseVersion := key.ReleaseVersion(nodePool.MachinePool)
	isSupported := capabilities.Supports(feature.NodePoolConditions, releaseVersion)
	if !isSupported {
		return naValue
	}

	if len(nodePool.MachinePool.Status.Conditions) > 0 {
		return formatCondition(string(nodePool.MachinePool.Status.Conditions[0].Type))
	}

	return naValue
}

func getAzureAutoscaling(nodePool nodepool.Nodepool, capabilities *feature.Service) string {
	releaseVersion := key.ReleaseVersion(nodePool.MachinePool)
	isSupported := capabilities.Supports(feature.Autoscaling, releaseVersion)
	if !isSupported {
		return naValue
	}

	minScaling, maxScaling := key.MachinePoolScaling(nodePool.MachinePool)
	if minScaling >= 0 && maxScaling >= 0 {
		return fmt.Sprintf("%d/%d", minScaling, maxScaling)
	}

	return naValue
}

func getAzureDescription(nodePool nodepool.Nodepool) string {
	description := key.MachinePoolName(nodePool.MachinePool)
	if len(description) < 1 {
		description = naValue
	}

	return description
}
