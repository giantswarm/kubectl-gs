package provider

import (
	"strings"

	infrastructurev1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
)

func getLatestCondition(conditions []infrastructurev1alpha2.CommonClusterStatusCondition) string {
	if len(conditions) < 1 {
		return "n/a"
	}

	condition := conditions[0].Condition

	return strings.ToUpper(condition)
}
