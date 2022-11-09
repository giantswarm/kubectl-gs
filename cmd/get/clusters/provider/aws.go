package provider

import (
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v2/internal/label"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
)

func GetAWSTable(clusterResource cluster.Resource) *metav1.Table {
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
		table.Rows = append(table.Rows, getAWSClusterRow(*c))
	case *cluster.Collection:
		for _, clusterItem := range c.Items {
			table.Rows = append(table.Rows, getAWSClusterRow(clusterItem))
		}
	}

	return table
}

func getAWSClusterRow(c cluster.Cluster) metav1.TableRow {
	if c.Cluster == nil || c.AWSCluster == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			c.AWSCluster.GetName(),
			output.TranslateTimestampSince(c.AWSCluster.CreationTimestamp),
			getLatestAWSCondition(c.AWSCluster.Status.Cluster.Conditions),
			c.AWSCluster.Labels[label.ReleaseVersion],
			getClusterServicePriority(c.Cluster),
			c.AWSCluster.Labels[label.Organization],
			c.AWSCluster.Spec.Cluster.Description,
		},
		Object: runtime.RawExtension{
			Object: c.AWSCluster,
		},
	}
}

func getLatestAWSCondition(conditions []infrastructurev1alpha3.CommonClusterStatusCondition) string {
	if len(conditions) < 1 {
		return naValue
	}

	return formatCondition(conditions[0].Condition)
}
