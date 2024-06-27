package provider

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v4/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v4/pkg/output"
)

func GetOpenStackTable(clusterResource cluster.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
		{Name: "Condition", Type: "string"},
		{Name: "Cluster Version", Type: "string"},
		{Name: "Preinstalled Apps Version", Type: "string"},
		{Name: "Service Priority", Type: "string"},
		{Name: "Organization", Type: "string"},
		{Name: "Description", Type: "string"},
	}

	switch c := clusterResource.(type) {
	case *cluster.Cluster:
		table.Rows = append(table.Rows, getOpenStackClusterRow(*c))
	case *cluster.Collection:
		for _, clusterItem := range c.Items {
			table.Rows = append(table.Rows, getOpenStackClusterRow(clusterItem))
		}
	}

	return table
}

func getOpenStackClusterRow(c cluster.Cluster) metav1.TableRow {
	if c.Cluster == nil {
		return metav1.TableRow{}
	}

	var clusterAppVersion string
	if c.ClusterApp != nil {
		clusterAppVersion = c.ClusterApp.Spec.Version
	}

	var defaultAppsAppVersion string
	if c.DefaultAppsApp != nil {
		defaultAppsAppVersion = c.DefaultAppsApp.Spec.Version
	}

	return metav1.TableRow{
		Cells: []interface{}{
			c.Cluster.GetName(),
			output.TranslateTimestampSince(c.Cluster.CreationTimestamp),
			getLatestCondition(c.Cluster.GetConditions()),
			clusterAppVersion,
			defaultAppsAppVersion,
			getClusterServicePriority(c.Cluster),
			getClusterOrganization(c.Cluster),
			getClusterDescription(c.Cluster),
		},
		Object: runtime.RawExtension{
			Object: c.Cluster,
		},
	}
}
