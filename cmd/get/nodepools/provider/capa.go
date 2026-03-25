package provider

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/giantswarm/kubectl-gs/v6/pkg/data/domain/nodepool"
)

func getCAPAAutoscaling(nodePool nodepool.Nodepool) string {
	if nodePool.EKSManagedMachinePool != nil {
		return getEKSManagedAutoscaling(nodePool)
	}

	if nodePool.CAPAMachinePool == nil {
		return naValue
	}

	minScaling, _, _ := unstructured.NestedInt64(nodePool.CAPAMachinePool.Object, "spec", "minSize")
	maxScaling, _, _ := unstructured.NestedInt64(nodePool.CAPAMachinePool.Object, "spec", "maxSize")

	return fmt.Sprintf("%d/%d", minScaling, maxScaling)
}

func getEKSManagedAutoscaling(nodePool nodepool.Nodepool) string {
	minScaling, _, _ := unstructured.NestedInt64(nodePool.EKSManagedMachinePool.Object, "spec", "scaling", "minSize")
	maxScaling, _, _ := unstructured.NestedInt64(nodePool.EKSManagedMachinePool.Object, "spec", "scaling", "maxSize")

	return fmt.Sprintf("%d/%d", minScaling, maxScaling)
}
