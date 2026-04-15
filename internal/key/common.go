package key

import (
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
)

func ClusterID(getter LabelsGetter) string {
	return getter.GetLabels()[label.Cluster]
}

func MachineDeploymentName(getter AnnotationsGetter) string {
	annotations := getter.GetAnnotations()
	if annotations == nil {
		return ""
	}

	return annotations[annotation.MachineDeploymentName]
}

func MachinePoolName(getter AnnotationsGetter) string {
	annotations := getter.GetAnnotations()
	if annotations == nil {
		return ""
	}

	return annotations[annotation.MachinePoolName]
}
