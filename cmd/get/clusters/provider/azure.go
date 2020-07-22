package provider

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/internal/label"
)

func GetAzureTable(resource runtime.Object) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "ID", Type: "string"},
		{Name: "Created", Type: "string", Format: "date-time"},
		{Name: "Condition", Type: "string"},
		{Name: "Release", Type: "string"},
		{Name: "Organization", Type: "string"},
		{Name: "Description", Type: "string"},
	}

	switch c := resource.(type) {
	case *capiv1alpha3.ClusterList:
		for _, cluster := range c.Items {
			table.Rows = append(table.Rows, getAzureClusterRow(&cluster))
		}

	case *capiv1alpha3.Cluster:
		table.Rows = append(table.Rows, getAzureClusterRow(c))
	}

	return table
}

func getAzureClusterRow(res *capiv1alpha3.Cluster) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			res.GetName(),
			res.CreationTimestamp.UTC(),
			getLatestAzureCondition(res),
			res.Labels[label.ReleaseVersion],
			res.Labels[label.Organization],
			getAzureClusterDescription(res),
		},
	}
}

func getAzureClusterDescription(res *capiv1alpha3.Cluster) string {
	description := "n/a"

	if desc, exists := res.Labels[label.Description]; exists {
		description = desc
	}

	return description
}

func getLatestAzureCondition(res *capiv1alpha3.Cluster) string {
	condition := ClusterStatusConditionCreating

	if res.Status.InfrastructureReady && res.Status.ControlPlaneInitialized && res.Status.ControlPlaneReady {
		condition = ClusterStatusConditionCreated
	}

	return formatCondition(condition)
}
