package provider

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v5/internal/feature"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/nodepool"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

func GetAzureTable(npResource nodepool.Resource, capabilities *feature.Service) *metav1.Table {
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
		table.Rows = append(table.Rows, getAzureNodePoolRow(*n, capabilities))
	case *nodepool.Collection:
		// Sort ASC by Cluster name.
		sort.Slice(n.Items, func(i, j int) bool {
			var iClusterName, jClusterName string

			if n.Items[i].MachinePool != nil && n.Items[i].MachinePool.Labels != nil {
				iClusterName = n.Items[i].MachinePool.Labels[capi.ClusterNameLabel]
			}
			if n.Items[j].MachinePool != nil && n.Items[j].MachinePool.Labels != nil {
				jClusterName = n.Items[j].MachinePool.Labels[capi.ClusterNameLabel]
			}

			return strings.Compare(iClusterName, jClusterName) > 0
		})

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
			nodePool.MachinePool.Labels[capi.ClusterNameLabel],
			output.TranslateTimestampSince(nodePool.MachinePool.CreationTimestamp),
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
	if releaseVersion == "" {
		return naValue
	}
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
	if releaseVersion == "" {
		return naValue
	}
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
