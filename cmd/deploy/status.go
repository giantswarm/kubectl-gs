package deploy

import (
	"context"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// checkKustomizations checks the health of all Kustomization resources
func (r *runner) checkKustomizations(ctx context.Context) (allReady bool, notReady []resourceInfo, suspended []resourceInfo, err error) {
	kustomizationList := &kustomizev1.KustomizationList{}

	err = r.ctrlClient.List(ctx, kustomizationList)
	if err != nil {
		return false, nil, nil, err
	}

	allReady = true
	notReady = []resourceInfo{}
	suspended = []resourceInfo{}

	for i := range kustomizationList.Items {
		kust := &kustomizationList.Items[i]

		// Check if suspended
		if kust.Spec.Suspend {
			suspended = append(suspended, resourceInfo{
				name:      kust.Name,
				namespace: kust.Namespace,
			})
			// Skip ready check for suspended kustomizations
			continue
		}

		// Check ready condition
		ready := false
		reason := ""
		for _, cond := range kust.Status.Conditions {
			if cond.Type == "Ready" {
				ready = (cond.Status == metav1.ConditionTrue)
				if !ready {
					reason = cond.Reason
				}
				break
			}
		}

		if !ready {
			allReady = false
			notReady = append(notReady, resourceInfo{
				name:      kust.Name,
				namespace: kust.Namespace,
				reason:    reason,
			})
		}
	}

	return allReady, notReady, suspended, nil
}
