package provider

import (
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/kubectl-gs/internal/label"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

func GetAzureTable(clusterResource cluster.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
		{Name: "Condition", Type: "string"},
		{Name: "Release", Type: "string"},
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
			getLatestAzureCondition(c.Cluster.GetConditions()),
			c.Cluster.Labels[label.ReleaseVersion],
			c.Cluster.Labels[label.Organization],
			getAzureClusterDescription(c.Cluster),
		},
		Object: runtime.RawExtension{
			Object: c.Cluster,
		},
	}
}

func getAzureClusterDescription(res *capiv1alpha3.Cluster) string {
	description := naValue

	annotations := res.GetAnnotations()
	if annotations != nil && annotations[annotation.ClusterDescription] != "" {
		description = annotations[annotation.ClusterDescription]
	}

	return description
}

func getLatestAzureCondition(conditions []capiv1alpha3.Condition) string {
	if len(conditions) < 1 {
		return naValue
	}

	return formatCondition(string(conditions[0].Type))
}
