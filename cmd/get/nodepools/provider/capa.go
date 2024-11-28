package provider

import (
	"fmt"

	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/nodepool"
)

func getCAPAAutoscaling(nodePool nodepool.Nodepool) string {
	if nodePool.EKSManagedMachinePool != nil {
		return getEKSManagedAutoscaling(nodePool)
	}

	minScaling := nodePool.CAPAMachinePool.Spec.MinSize
	maxScaling := nodePool.CAPAMachinePool.Spec.MaxSize

	return fmt.Sprintf("%d/%d", minScaling, maxScaling)
}

func getEKSManagedAutoscaling(nodePool nodepool.Nodepool) string {
	minScaling := nodePool.EKSManagedMachinePool.Spec.Scaling.MinSize
	maxScaling := nodePool.EKSManagedMachinePool.Spec.Scaling.MaxSize

	return fmt.Sprintf("%d/%d", *minScaling, *maxScaling)
}
