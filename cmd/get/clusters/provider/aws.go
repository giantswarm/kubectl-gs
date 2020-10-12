package provider

import (
	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/v2/pkg/apis/infrastructure/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/internal/label"
)

func GetAWSTable(resource runtime.Object) *metav1.Table {
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
	case *infrastructurev1alpha2.AWSClusterList:
		for _, cluster := range c.Items {
			table.Rows = append(table.Rows, getAWSClusterRow(&cluster))
		}

	case *infrastructurev1alpha2.AWSCluster:
		table.Rows = append(table.Rows, getAWSClusterRow(c))
	}

	return table
}

func getAWSClusterRow(res *infrastructurev1alpha2.AWSCluster) metav1.TableRow {
	return metav1.TableRow{
		Cells: []interface{}{
			res.GetName(),
			res.CreationTimestamp.UTC(),
			getLatestAWSCondition(res.Status.Cluster.Conditions),
			res.Labels[label.ReleaseVersion],
			res.Labels[label.Organization],
			res.Spec.Cluster.Description,
		},
	}
}

func getLatestAWSCondition(conditions []infrastructurev1alpha2.CommonClusterStatusCondition) string {
	if len(conditions) < 1 {
		return naValue
	}

	return formatCondition(conditions[0].Condition)
}
