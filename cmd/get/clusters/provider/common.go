package provider

import (
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

const (
	naValue = "n/a"
)

func GetCommonClusterTable(clusterResource cluster.Resource) *metav1.Table {
	// Creating a custom table resource.
	table := &metav1.Table{}

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string"},
		{Name: "Age", Type: "string", Format: "date-time"},
		{Name: "Phase", Type: "string"},
		{Name: "Release", Type: "string"},
		{Name: "Service Priority", Type: "string"},
		{Name: "Organization", Type: "string"},
		{Name: "Description", Type: "string"},
	}

	switch c := clusterResource.(type) {
	case *cluster.Cluster:
		table.Rows = append(table.Rows, getCommonClusterRow(*c))
	case *cluster.Collection:
		for _, clusterItem := range c.Items {
			table.Rows = append(table.Rows, getCommonClusterRow(clusterItem))
		}
	}

	return table
}

func formatCondition(condition string) string {
	return strings.ToUpper(condition)
}

func getClusterDescription(obj *unstructured.Unstructured) string {
	annotations := obj.GetAnnotations()
	if annotations != nil && annotations[annotation.ClusterDescription] != "" {
		return annotations[annotation.ClusterDescription]
	}
	return naValue
}

func getClusterOrganization(obj *unstructured.Unstructured) string {
	labels := obj.GetLabels()
	if organizationLabel := labels[label.Organization]; organizationLabel != "" {
		return organizationLabel
	}
	return naValue
}

func getClusterServicePriority(obj *unstructured.Unstructured) string {
	labels := obj.GetLabels()
	if servicePriorityLabel := labels[label.ServicePriority]; servicePriorityLabel != "" {
		return servicePriorityLabel
	}
	return naValue
}

func getCommonClusterRow(c cluster.Cluster) metav1.TableRow {
	if c.Cluster == nil {
		return metav1.TableRow{}
	}

	phase, _, _ := unstructured.NestedString(c.Cluster.Object, "status", "phase")

	return metav1.TableRow{
		Cells: []interface{}{
			c.Cluster.GetName(),
			output.TranslateTimestampSince(c.Cluster.GetCreationTimestamp()),
			phase,
			c.Cluster.GetLabels()[label.ReleaseVersion],
			getClusterServicePriority(c.Cluster),
			c.Cluster.GetLabels()[label.Organization],
			getClusterDescription(c.Cluster),
		},
		Object: runtime.RawExtension{
			Object: c.Cluster,
		},
	}
}
