package provider

import (
	"strings"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

func getLatestV4Condition(conditions []providerv1alpha1.StatusClusterCondition) string {
	if len(conditions) < 1 {
		return "n/a"
	}

	condition := conditions[0].Type

	return strings.ToUpper(condition)
}

func getLatestV5Condition(conditions []infrastructurev1alpha2.CommonClusterStatusCondition) string {
	if len(conditions) < 1 {
		return "n/a"
	}

	condition := conditions[0].Condition

	return strings.ToUpper(condition)
}
