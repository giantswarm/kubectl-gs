package provider

import (
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	naValue = "n/a"
)

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
