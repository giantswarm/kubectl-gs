package provider

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/kubectl-gs/v5/pkg/output"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"

	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/cluster"
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

func getLatestCondition(conditions []capi.Condition) string {
	if len(conditions) < 1 {
		return naValue
	}

	return formatCondition(string(conditions[0].Type))
}

func getClusterDescription(res *capi.Cluster) string {
	description := naValue

	annotations := res.GetAnnotations()
	if annotations != nil && annotations[annotation.ClusterDescription] != "" {
		description = annotations[annotation.ClusterDescription]
	}

	return description
}

func getClusterOrganization(res *capi.Cluster) string {
	organization := naValue

	if organizationLabel := res.Labels[label.Organization]; organizationLabel != "" {
		organization = organizationLabel
	}

	return organization
}

func getClusterServicePriority(res *capi.Cluster) string {
	servicePriority := naValue

	if servicePriorityLabel := res.Labels[label.ServicePriority]; servicePriorityLabel != "" {
		servicePriority = servicePriorityLabel
	}

	return servicePriority
}

func getCommonClusterRow(c cluster.Cluster) metav1.TableRow {
	if c.Cluster == nil {
		return metav1.TableRow{}
	}

	return metav1.TableRow{
		Cells: []interface{}{
			c.Cluster.GetName(),
			output.TranslateTimestampSince(c.Cluster.CreationTimestamp),
			c.Cluster.Status.Phase,
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
