package provider

import (
	"strings"
)

const (
	ClusterStatusConditionCreated  = "Created"
	ClusterStatusConditionCreating = "Creating"

	ClusterStatusConditionDeleted  = "Deleted"
	ClusterStatusConditionDeleting = "Deleting"

	ClusterStatusConditionUpdated  = "Updated"
	ClusterStatusConditionUpdating = "Updating"
)

func formatCondition(condition string) string {
	return strings.ToUpper(condition)
}
