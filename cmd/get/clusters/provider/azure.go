package provider

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v2/internal/label"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
)

func GetAzureTable(clusterResource cluster.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
		{Name: "Condition", Type: "string"},
		{Name: "Release", Type: "string"},
		{Name: "Service Priority", Type: "string"},
		{Name: "Organization", Type: "string"},
		{Name: "Description", Type: "string"},
	}

	switch c := clusterResource.(type) {
	case *cluster.Cluster:
		table.Rows = append(table.Rows, getAzureClusterRow(*c))
	case *cluster.Collection:
		for _, clusterItem := range c.Items {
			table.Rows = append(table.Rows, getAzureClusterRow(clusterItem))
		}
	}

	return table
}

func getAzureClusterRow(c cluster.Cluster) metav1.TableRow {
	if c.Cluster == nil || c.AzureCluster == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			c.Cluster.GetName(),
			output.TranslateTimestampSince(c.Cluster.CreationTimestamp),
			getLatestCondition(c.Cluster.GetConditions()),
			c.Cluster.Labels[label.ReleaseVersion],
			getClusterServicePriority(c.Cluster),
			c.Cluster.Labels[label.Organization],
			getClusterDescription(c.Cluster),
		},
		Object: runtime.RawExtension{
			Object: c.Cluster,
		},
	}
}
